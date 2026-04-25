include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-echarging
PKG_VERSION:=1.0.0
PKG_RELEASE:=4

PKG_MAINTAINER:=Quiet
PKG_LICENSE:=MIT

include $(INCLUDE_DIR)/package.mk

define Package/$(PKG_NAME)
  SECTION:=luci
  CATEGORY:=LuCI
  SUBMENU:=3. Applications
  TITLE:=LuCI support for echarging
  PKGARCH:=all
endef

define Package/$(PKG_NAME)/description
宿舍电费自动查询、预测与低余额提醒
endef

define Build/Prepare
endef

define Build/Compile
endef

define Package/$(PKG_NAME)/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) ./files/usr/bin/echarging $(1)/usr/bin/echarging
	$(INSTALL_BIN) ./files/usr/bin/echarging-uci2json $(1)/usr/bin/echarging-uci2json

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/etc/config/echarging $(1)/etc/config/echarging

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/etc/init.d/echarging $(1)/etc/init.d/echarging

	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/controller
	$(INSTALL_DATA) ./luasrc/controller/echarging.lua $(1)/usr/lib/lua/luci/controller/echarging.lua

	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/model/cbi
	$(INSTALL_DATA) ./luasrc/model/cbi/echarging.lua $(1)/usr/lib/lua/luci/model/cbi/echarging.lua

	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/view/echarging
	$(INSTALL_DATA) ./luasrc/view/echarging/status.htm $(1)/usr/lib/lua/luci/view/echarging/status.htm

	$(INSTALL_DIR) $(1)/usr/share/luci/menu.d
	$(INSTALL_DATA) ./root/usr/share/luci/menu.d/luci-app-echarging.json $(1)/usr/share/luci/menu.d/luci-app-echarging.json

	$(INSTALL_DIR) $(1)/usr/share/rpcd/acl.d
	$(INSTALL_DATA) ./root/usr/share/rpcd/acl.d/luci-app-echarging.json $(1)/usr/share/rpcd/acl.d/luci-app-echarging.json
endef

define Package/$(PKG_NAME)/conffiles
/etc/config/echarging
endef

define Package/$(PKG_NAME)/postinst
#!/bin/sh
[ -n "$$IPKG_INSTROOT" ] && exit 0

OLD_DB="/var/lib/echarging/echarging.db"
NEW_DB="/etc/echarging/echarging.db"

mkdir -p /etc/echarging

if command -v uci >/dev/null 2>&1; then
	current_db="$$(uci -q get echarging.global.db_path)"
	if [ -z "$$current_db" ] || [ "$$current_db" = "$$OLD_DB" ]; then
		uci set echarging.global.db_path="$$NEW_DB"
		uci commit echarging
	fi
fi

if [ -f "$$OLD_DB" ] && [ ! -f "$$NEW_DB" ]; then
	cp "$$OLD_DB" "$$NEW_DB"
fi

exit 0
endef

$(eval $(call BuildPackage,$(PKG_NAME)))
