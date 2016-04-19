local runtime = require("gws")
local stat = require("stat")
local time = require("time")

local message = "test message"
local sleep = "10ms"
local dialLimit = 100
local start = time.now(time.s)

if runtime.isMaster() then
    runtime.on("exit", exit)

    stat.new("duration",       stat.abs())
    stat.new("dials",          stat.abs())
    stat.new("errors_send",    stat.abs())
    stat.new("errors_receive", stat.abs())
    stat.new("threads",        stat.abs())
    stat.new("messages_in",    stat.per("1s"), stat.abs())
    stat.new("messages_out",   stat.per("1s"), stat.abs())
    stat.new("delay",          stat.avg())

    for i = 0, 100 do
        runtime.fork()
    end
else
    setup()
end

local function exit()
    stat.add("duration", time.now(time.s) - start)
    --    print(stat.pretty())
end

local id;
local attempts = 0;

local function setup()
    stat.add("threads", 1)
    id = runtime:get("id")
    runtime.on("connect", connect)
    runtime.on("teardown", teardown)
    runtime.nextTick(tick)
end

local function connect()
    stat.add("dials", 1)
    if attempts+1 > dialLimit then
        runtime.exit()
    else
        attempts = attempts + 1
    end
end

local function teardown()
    print("thread teardown", id, attempts)
end

local function tick(thread)
    local timeout;
    local err = runtime.send(message)
    if err ~= nil then
        print("send error:", err)
        stat.add("errors_send", 1)
        return runtime.setTimeout(tick, sleep)
    end
    stat.add("messages_out", 1)

    local start = time.now(time.ms)
    local _, err = thread:receive()
    if err ~= nil then
        print("receive error:", err)
        stat.add("errors_receive", 1)
    else
        stat.add("delay", time.now(time.ms) - start)
        stat.add("messages_in", 1)
    end

    runtime.setTimeout(tick, 0)
end

