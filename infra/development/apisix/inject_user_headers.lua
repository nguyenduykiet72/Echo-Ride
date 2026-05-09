-- Decodes the Bearer JWT and injects user identity headers for downstream
-- services. APISIX jwt-auth has already verified the signature + exp by the
-- time this runs in the rewrite phase.
return function(conf, ctx)
    local jwt_lib = require("resty.jwt")
    local h = ngx.req.get_headers()["authorization"] or ""
    local token = h:gsub("^Bearer%s+", "")
    if token == "" then return end
    local jwt_obj = jwt_lib:load_jwt(token)
    if not jwt_obj or not jwt_obj.payload then return end
    if jwt_obj.payload.sub then
        ngx.req.set_header("X-User-Id", jwt_obj.payload.sub)
    end
    if jwt_obj.payload.role then
        ngx.req.set_header("X-User-Role", jwt_obj.payload.role)
    end
end
