include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-echarging
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

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

	$(INSTALL_DIR) $(1)/usr/share/luci/menu.d
	$(INSTALL_DATA) ./root/usr/share/luci/menu.d/luci-app-echarging.json $(1)/usr/share/luci/menu.d/luci-app-echarging.json

	$(INSTALL_DIR) $(1)/usr/share/rpcd/acl.d
	$(INSTALL_DATA) ./root/usr/share/rpcd/acl.d/luci-app-echarging.json $(1)/usr/share/rpcd/acl.d/luci-app-echarging.json
endef

define Package/$(PKG_NAME)/conffiles
/etc/config/echarging
endef

$(eval $(call BuildPackage,$(PKG_NAME)))
