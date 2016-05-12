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

    stat.new("duration",       stat.abs())
    stat.new("dials",          stat.abs())
    stat.new("errors_send",    stat.abs())
    stat.new("errors_receive", stat.abs())
    stat.new("threads",        stat.abs())
    stat.new("messages_in",    stat.per("1s"), stat.abs())
    stat.new("messages_out",   stat.per("1s"), stat.abs())
    stat.new("delay",          stat.avg())

    for i = 0, 127 do
        runtime.fork()
    end

    return
end

-- This is a fork part

local limit = 1000

ws.connect({ url = runtime.get("url") }, function(err, conn)
    if (err ~= nil) then
        print("could not connect: ", err)
        return
    end

    print("connected to ", runtime.get("url"))
    conn.send("hello, my lord!", function(err)
        if err ~= nil then
            print("async send error: ", err)
        else
            print("async send ok")
        end
    end)

    time.setTimeout(1500, function()
        print("timeout!")
        local err = conn.send("SYNC MESSAGE");
        if err ~= nil then
            print("sync send error:", err);
        else
            print("sync send ok");
        end
    end)

    local cnt = 0
    conn.listen(function(err, msg)
        if err ~= nil then
            print("receive error: ", err)
        else
            local text = "received: %s (%d)"
            print(text.format(text, msg, cnt))

--            if cnt > limit+1 then
--                print("will close now!")
--                conn.close()
--            end
            cnt = cnt + 1
        end
    end)

    conn.on("close", function()
        print("conn closed")
    end)

    for i = 0, limit do
        local err = conn.send(string.format("message %d", i))
        if err ~= nil then
            print(i, err)
            return
        end
    end
end)

runtime.on("exit", function()
    print("thread is exiting now.. bye!")
    --    print(stat.pretty())
end)