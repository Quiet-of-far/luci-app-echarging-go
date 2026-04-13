module("luci.controller.echarging", package.seeall)

local http = require("luci.http")
local jsonc = require("luci.jsonc")
local sys = require("luci.sys")

function index()
	entry({"admin", "services", "echarging"}, cbi("echarging"), _("电量监控"), 90)
	entry({"admin", "services", "echarging", "status"}, call("action_status")).leaf = true
	entry({"admin", "services", "echarging", "query_now"}, call("action_query_now")).leaf = true
	entry({"admin", "services", "echarging", "test_email"}, call("action_test_email")).leaf = true
	entry({"admin", "services", "echarging", "test_wxpusher"}, call("action_test_wxpusher")).leaf = true
end

function action_status()
	render_command("/usr/bin/echarging -config /var/etc/echarging.json status 2>&1", true)
end

function action_query_now()
	render_command("/usr/bin/echarging -config /var/etc/echarging.json query-now 2>&1", true)
end

function action_test_email()
	render_command("/usr/bin/echarging -config /var/etc/echarging.json test-notify --channel email 2>&1", true)
end

function action_test_wxpusher()
	render_command("/usr/bin/echarging -config /var/etc/echarging.json test-notify --channel wxpusher 2>&1", true)
end

function render_command(command, regenerate_config)
	if regenerate_config then
		sys.call("/usr/bin/echarging-uci2json >/dev/null 2>&1")
	end

	local output = sys.exec(command)
	local payload = jsonc.parse(output)

	http.prepare_content("application/json")
	if payload then
		http.write_json(payload)
		return
	end

	http.write_json({
		status = "error",
		message = output ~= "" and output or "命令执行失败"
	})
end
