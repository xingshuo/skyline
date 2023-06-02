local Skynet = require "skynet"
local Cluster = require "skynet.cluster"
require "skynet.manager"

local loClusName = "skynet_server"

local CMD = {}

function CMD.Response(ts, map)
    print("get rsp ts:", ts)
    print("get rsp map:")
    for k,v in pairs(map) do
        print(k, " : ", v)
    end
end

local bigData = string.rep("x", 80000)

function CMD.MultiResponse(data)
    print("get multi rsp data len: ", #data)
    assert(bigData == data, "data error")
end

Skynet.start(function()
    Skynet.dispatch("lua", function(_,_, command, ...)
        local f = CMD[command]
        Skynet.ret(Skynet.pack(f(...)))
    end)
    Cluster.open(loClusName)
    Skynet.register(".routersender")
    Skynet.fork(function ()
        Cluster.send("go_server", "routeragent", "lua", "Request", loClusName)
        Skynet.sleep(150)
        Cluster.send("go_server", "routeragent", "lua", "MultiRequest", loClusName, bigData)
    end)
end)