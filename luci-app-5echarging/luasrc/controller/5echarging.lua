module("luci.controller.5echarging", package.seeall)

local http = require("luci.http")
local jsonc = require("luci.jsonc")
local sys = require("luci.sys")

function index()
	entry({"admin", "services", "5echarging"}, cbi("5echarging"), _("抑疑电止"), 90)
	entry({"admin", "services", "5echarging", "status"}, call("action_status")).leaf = true
	entry({"admin", "services", "5echarging", "query_now"}, call("action_query_now")).leaf = true
	entry({"admin", "services", "5echarging", "test_email"}, call("action_test_email")).leaf = true
end

function action_status()
	render_command("/usr/bin/5echarging -config /var/etc/5echarging.json status", true)
end

function action_query_now()
	render_command("/usr/bin/5echarging -config /var/etc/5echarging.json query-now", true)
end

function action_test_email()
	render_command("/usr/bin/5echarging -config /var/etc/5echarging.json test-notify", true)
end

function render_command(command, regenerate_config)
	if regenerate_config then
		sys.call("/usr/bin/5echarging-uci2json >/dev/null 2>&1")
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
