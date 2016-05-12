local time = require("time")
local runtime = require("runtime")
local ws = require("ws")

ws.connect({ url = runtime.get("url") }, function(err, conn)
    if (err ~= nil) then
        print("could not connect: ", err)
        return
    end

    print("connected to ", runtime.get("url"))

    time.setTimeout(1000, conn.close)
end)