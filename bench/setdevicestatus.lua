request = function()
    path = "/status/devicetype/deviceid"
    wrk.headers["Content-Type"] = "application/json; charset=utf-8"
    wrk.body = "{\"user_id\":\"userid\",\"status\":\"online\"}"
    return wrk.format("PUT", path)
end
