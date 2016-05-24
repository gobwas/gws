local stat = require("stat")
local runtime = require("runtime")
local ws = require("ws")
local time = require("time")

if runtime.isMaster() then
    local start = time.now(time.ms)

    stat.new("threads",     stat.abs())
    stat.new("duration",    stat.abs({ measure = "ms" }))
    stat.new("connections", stat.abs())
    stat.new("messages",    stat.abs())

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
    local addr = runtime.get("listen")

    server.listen(addr, function(conn)
        stat.add("connections", 1)
        conn.listen(function()
            stat.add("messages", 1)
        end)
    end)

    server.on("error", function(err)
        print("error: ", err)
    end)

    server.on("listening", function()
        print("listening: ", addr)
    end)
end