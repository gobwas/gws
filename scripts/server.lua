local stat = require("stat")
local runtime = require("runtime")
local ws = require("ws")

if runtime.isMaster() then
    stat.new("threads",    stat.abs())
    stat.new("duration",   stat.abs({ measure = "ms" }))

    for i = 0, (runtime.numCPU-1) do
        runtime.fork()
        stat.add("threads", 1)
    end

    runtime.on("exit", function()
        print("bye!")
        stat.add("duration", time.now(time.ms) - start)
    end)
else
    local server = ws.createServer()

    server.listen(":8083", function(conn)
        print("got conn")
        conn.send("hi there")
    end)

    server.on("error", function(err)
        print("error: ", err)
    end)

    server.on("listening", function()
        print("listening")
    end)
end