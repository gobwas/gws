local stat = require("gws.stat")
local time = require("gws.time")

local message = "test message"

function main()
    stat.abs("dials")
    stat.abs("errors_send")
    stat.abs("errors_receive")
    stat.abs("threads")
    stat.per("request", "1s")
    stat.avg("delay")
end

function setup(thread)
    thread:set("id", stat.add("threads", 1))
    thread:set("calls", 0)
    table.insert(threads, thread)
end

function teardown(thread)
    print("thread teardown", thread:get("id"))
end

-- reconnect return true for thread's connection should be alive,
-- or false for connection should stay closed, and thread eventually die
-- reconnect called on connections was closed (by server, or by thread.close())
function reconnect(thread)
    stat.add("dials", 1)
    local calls = thread:get("calls")
    thread:set("calls", calls+1)
    thread:sleep("1s")
    return true
end

function tick(thread)
    stat.add("request", 1)

    local err = thread.send(message)
    if err ~= "" then
        stat.add("errors_send", 1)
        print("send error", err)
        thread.sleep("1s")
        return
    end

    local start = time.now(time.ms)
    local _, err = thread.receive()
    if err ~= "" then
        stat.add("errors_receive", 1)
        print("receive error", err)
    else
        stat.add("delay", time.now(time.ms) - start)
    end

    thread.sleep("1000ms")
end