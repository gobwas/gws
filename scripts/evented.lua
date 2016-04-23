local stat = require("stat")
local time = require("time")
local runtime = require("runtime")
local ws = require("ws")

local start = time.now(time.s)

ws.connect({ url = runtime.get("url") }, function(err, conn)
    if (err ~= nil) then
        print("could not connect: ", err)
    else
        print("connected!")
        conn.send("hello, my lord!", function(err)
            if err ~= nil then
                print("send error: ", err)
            end
        end)

        time.setTimeout(1500, function()
            print("timeout!")
            print("sync send: ", conn.send("DDD"))
        end)

        local cnt = 0
        conn.listen(function(err, msg)
            if err ~= nil then
                print("receive error: ", err)
            else
                cnt = cnt + 1
                print("received: ", msg)
                if cnt > 1 then
                    print("will close now!")
                    conn.close()
                end
            end
        end)

        conn.on("close", function()
            print("conn closed")
        end)
    end
end)



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

    for i = 0, 99 do
        runtime.fork()
    end

    return
end

-- This is a fork part

ws.connect({ url = runtime.get("url") }, function(err, conn)
--    conn.on("message", function(message)
--        print("got message", message)
--    end)
--
--    conn.send("hello", function(err)
--        print("sent", err)
--    end)
--
--    conn.close(function(err)
--        print("closed", err)
--    end)
end)
