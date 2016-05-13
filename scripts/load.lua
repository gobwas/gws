local stat = require("stat")
local time = require("time")
local runtime = require("runtime")
local ws = require("ws")

local start = time.now(time.ms)

if runtime.isMaster() then
    stat.new("err",      stat.abs())
    stat.new("threads",  stat.abs())
    stat.new("sent",     stat.per("1s"), stat.abs())
    stat.new("delay",    stat.avg({ measure = "ms" }))
    stat.new("duration", stat.abs({ measure = "ms" }))

    for i = 0, (runtime.numCPU-1) do
        runtime.fork()
        stat.add("threads", 1)
    end

    runtime.on("exit", function()
        print("bye!")
        stat.add("duration", time.now(time.ms) - start)
    end)
else
    --[[
        runtime exports:
            url      -u flag value
            headers  -h flag value
            id       incremental identifier of the fork
    ]]
    ws.connect({ url = runtime.get("url") }, function(err, conn)
        if (err ~= nil) then
            print("could not connect: ", err)
            return
        else
            -- if you want to get the maximum throughput in your tests
            -- and not need to receive answers from the server under test,
            -- it is preferred to be synchronous.
            while true do
                local start = time.now(time.ms)
                if (conn.send("hello") ~= nil) then
                    stat.add("err", 1)
                else
                    stat.add("sent", 1)
                    stat.add("delay", time.now(time.ms) - start)
                end
            end
        end
    end)

    runtime.on("exit", function()
        print("bye!")
    end)
end
