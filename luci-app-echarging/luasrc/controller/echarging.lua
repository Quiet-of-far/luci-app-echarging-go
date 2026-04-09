module("luci.controller.echarging", package.seeall)

function index()
	entry({"admin", "services", "echarging"}, cbi("echarging"), _("电费监控"), 90)
end
