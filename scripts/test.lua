local stat = require("gws.stat")
local time = require("gws.time")

local message = "test message"
local sleep = "0"
local dialLimit = 100
local start = time.now(time.s)

function main()
    stat.new("duration",       stat.abs({measure="sec"}))
    stat.new("dials",          stat.abs())
    stat.new("errors_send",    stat.abs())
    stat.new("errors_receive", stat.abs())
    stat.new("threads",        stat.abs())
    stat.new("messages_in",    stat.per("1s", {measure="rps"}), stat.abs())
    stat.new("messages_out",   stat.per("1s", {measure=""}), stat.abs())
    stat.new("delay",          stat.avg({measure="msec"}))
end

function done()
    stat.add("duration", time.now(time.s) - start)
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
    local tags = {
        thread=thread:get("id")
    }

    local err = thread:send(message)
    if err ~= nil then
        print("send error:", err)
        stat.add("errors_send", 1, tags)
        thread:sleep(sleep)
        return
    else
        stat.add("messages_out", 1, tags)
    end

    local start = time.now(time.ms)
    local _, err = thread:receive()
    if err ~= nil then
        print("receive error:", err)
        stat.add("errors_receive", 1, tags)
    else
        stat.add("delay", time.now(time.ms) - start, tags)
        stat.add("messages_in", 1, tags)
    end

    thread:sleep(sleep)
end