local stat = require("gws.stat")
local time = require("gws.time")

local message = "test message"
local sleep = "0"
local dialLimit = 100
local start = time.now("seconds")

function main()
    stat.abs("duration")
    stat.abs("dials")
    stat.abs("errors_send")
    stat.abs("errors_receive")
    stat.abs("threads")
    stat.per("messages_in", "1s")
    stat.per("messages_out", "1s")
    stat.avg("delay")
end

function done()
    -- todo here not 5s
    stat.add("duration", time.now("seconds") - start)
    print(stat.pretty())
end

function setup(thread, id)
    stat.add("threads", 1)
    thread:set("id", id)
    thread:set("attempts", 0)
end

function teardown(thread)
    print("thread teardown", thread:get("id"))
end

-- reconnect return true for thread's connection should be alive,
-- or false for connection should stay closed, and thread eventually die
-- reconnect called on connections was closed (by server, or by thread.close())
function reconnect(thread)
    stat.add("dials", 1)

    local attempts = thread:get("attempts")
    if attempts+1 > dialLimit then
        return false
    end

    thread:set("attempts", attempts +1)
    thread:sleep("1s")

    return true
end

function tick(thread)
    local err = thread:send(message)
    if err ~= nil then
        print("send error:", err)
        stat.add("errors_send", 1)
        thread:sleep(sleep)
        return
    else
        stat.add("messages_out", 1)
    end

    local start = time.now(time.milliseconds)
    local _, err = thread:receive()
    if err ~= nil then
        print("receive error:", err)
        stat.add("errors_receive", 1)
    else
        stat.add("delay", time.now(time.milliseconds) - start)
        stat.add("messages_in", 1)
    end

    thread:sleep(sleep)
end