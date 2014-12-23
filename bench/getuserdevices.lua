request = function()
    path = "/users/userid"
    wrk.headers["Content-Type"] = "application/json; charset=utf-8"
    return wrk.format("GET", path)
end
