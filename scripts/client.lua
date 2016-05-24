local stat = require("stat")
local time = require("time")
local runtime = require("runtime")
local ws = require("ws")

local start = time.now(time.ms)

if runtime.isMaster() then
    stat.new("send_err",   stat.abs())
    stat.new("send_ok",    stat.abs())
    stat.new("recv_err",   stat.abs())
    stat.new("recv_ok",    stat.abs())
    stat.new("threads",    stat.abs())
    stat.new("recv_delay", stat.avg({ measure = "ms" }))
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
    local function send(conn)
        conn.send("hello", function(err)
            if (err ~= nil) then
                stat.add("err", 1)
            else
                stat.add("sent", 1)
                stat.add("delay", time.now(time.ms) - start)
            end
        end)
    end

    --[[
        runtime exports:
            url      -u flag value
            headers  -h flag value
            id       incremental identifier of the fork
    ]]
    ws.connect({ url = runtime.get("url") }, function(err, conn)
        if (err ~= nil) then
            print("could not connect: ", err)
        else
            print("connect ok")

            local start;
            local send = function()
                start = time.now(time.ms)
                conn.send("hello", function(err)
                    if (err ~= nil) then
                        stat.add("send_err", 1)
                        print("send error: ", err)
                    else
                        stat.add("send_ok", 1)
                        print("send ok")
                    end
                end)
            end

            conn.listen(function(err, msg)
                if err ~= nil then
                    stat.add("recv_err", 1)
                    print("receive error: ", err)
                else
                    stat.add("recv_ok", 1)
                    stat.add("recv_delay", time.now(time.ms) - start)
                    print(string.format("receive ok: %s (%d)", msg))
                end

                time.setTimeout(100, send)
            end)

            conn.on("close", function()
                print("conn closed")
            end)

            send()
        end
    end)

    runtime.on("exit", function()
        print("bye!")
    end)
end
