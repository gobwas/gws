local m = require("client")

local counter = 1
local threads = {}

function setup(thread)
    thread:set("id", counter)
    thread.id = counter

    table.insert(threads, thread)
    counter = counter + 1

    for variable = 0, 1, 1 do
--local message = "message %d"
--thread.send(message:format(variable))
        local json = '{"id":0,"method":"page","params":{"page_id":"21753","search_uid":"2158983901449053434","location":"https://e.mail.ru/messages/inbox","stat_id":"1510301","agent":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/48.0.2564.109 Safari/537.36"},"jsonrpc":"2.0"}'
        thread.send(json)

        local resp, err = thread.receive()
        if err ~= "" then
            print("receive error:", err)
--        else
--            print("receive resp:", resp)
        end
    end
end
