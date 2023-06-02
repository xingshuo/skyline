local Skynet = require "skynet"

Skynet.start(function()
    Skynet.newservice("routersender")    
    Skynet.exit()
end)