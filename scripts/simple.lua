local stat = require("stat")
local time = require("time")
local runtime = require("runtime")
local ws = require("ws")

local start = time.now(time.s)

if runtime.isMaster() then
    runtime.on("exit", function()
        print("exiting now.. bye!")
        stat.add("duration", time.now(time.s) - start)
        --    print(stat.pretty())
    end)

    stat.new("err",     stat.abs())
    stat.new("threads", stat.abs())
    stat.new("sent",    stat.per("1s"), stat.abs())
    stat.new("delay",   stat.avg())

    for i = 0, (runtime.numCPU-1) do
        runtime.fork()
        stat.add("threads", 1)
    end

    return
end

-- This is a fork part

local function work(conn)
    local start = time.now(time.ms)
    conn.send("hello", function(err)
        if (err ~= nil) then
            stat.add("err", 1)
        else
            stat.add("sent", 1)
            stat.add("delay", time.now(time.ms) - start)
        end
        work(conn);
    end)
end

ws.connect({ url = runtime.get("url") }, function(err, conn)
    if (err ~= nil) then
        print("could not connect: ", err)
        return
    end

    print("connected to ", runtime.get("url"))

    work(conn)

    time.setInterval(1000, function()
        print("interval")
    end)

--    while true do
--        conn.send("hello")
--        stat.add("sent", 1)
--    end
end)

runtime.on("exit", function()
--    print("thread is exiting now.. bye!")
    --    print(stat.pretty())
end)