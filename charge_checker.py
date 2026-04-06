import requests
import time
from mcdreforged.api.all import *

# 插件元信息
__plugin_meta__ = {
    'id': 'charge_checker',
    'version': '1.1.2',  # 修正版本号
    'name': 'charge_checker',
    'author': 'Quiet-of-far (modified by Gemini)',
    'description': '一个MCDR插件，用于每小时检查电费，并支持玩家自助查询任意房间电费。'
}

# 将错误信息字典定义为全局常量，方便复用
ERROR_TRANSLATION = {
    'api_error': 'API返回错误',
    'http_error': 'HTTP请求错误',
    'request_error': '网络请求失败',
    'parse_error': '响应解析失败'
}


def get_balance_from_api(building: str, room: str) -> tuple[str, str]:
    """
    请求电费查询API，获取指定楼栋和房间的电费余额。
    """
    url = "http://202.192.240.231/scp-api/electricity-recharge/getCurrentRemaining"
    payload = {
        'userTypeID': 1,
        'building': building,
        'room': room
    }
    try:
        response = requests.post(url, data=payload, timeout=10)
        if response.status_code == 200:
            try:
                result = response.json()
                if result.get('success'):
                    balance_str = result.get('data')
                    return 'success', str(balance_str)
                else:
                    error_message = result.get('message', '未知的API错误')
                    return 'api_error', error_message
            except ValueError:
                return 'parse_error', f"响应解析失败. 原始返回: {response.text}"
        else:
            return 'http_error', f"HTTP请求失败，状态码 {response.status_code}. 返回: {response.text}"
    except requests.exceptions.RequestException as e:
        return 'request_error', f"请求过程中发生错误: {e}. 请检查网络或目标服务器是否可达。"


def check_server_electricity(server: ServerInterface):
    """
    检查服务器默认房间（44栋-1207室）的电费。
    """
    default_building = '44'
    default_room = '1207'
    status, data = get_balance_from_api(default_building, default_room)

    if status == 'success':
        balance_str = data
        server.logger.info(f"成功检查服务器电费. 当前余额: {balance_str}")
        try:
            balance = float(balance_str)
            if balance < 15:
                server.logger.warning(f"电费余额过低 ({balance}). 正在通知玩家...")
                server.execute('title @a title {"text":"服务器电费低于安全阈值","color":"red"}')
                server.execute('title @a subtitle {"text":"请尽快通知管理员！"}')
                server.execute('say "服务器电费低于安全阈值"')
                server.execute('say "请尽快通知管理员！"')
        except (ValueError, TypeError) as e:
            server.logger.error(f"无法将余额转换为数字: {e}. 余额字符串为: '{balance_str}'")
    else:
        server.logger.error(f"检查服务器电费失败 ({status}): {data}")


@new_thread('ChargeCheckerWorker')
def hourly_check_worker(server: ServerInterface):
    """一个在独立线程中运行的守护进程，每小时检查一次电费。"""
    while True:
        check_server_electricity(server)
        time.sleep(3600)


def on_load(server: ServerInterface, old):
    """插件加载时运行，负责注册指令和启动守护进程。"""

    # 处理玩家自助查询指令: !!charge check <building> <room>
    def manual_user_query(src: CommandSource, context: dict):
        building = context['building']
        room = context['room']
        
        src.reply(f"正在查询，请稍候...")

        @new_thread(f'ManualCheck-{building}-{room}')
        def do_check():
            status, data = get_balance_from_api(building, room)
            if status == 'success':
                src.reply(f"§a查询成功！\n当前余额：§6{data}§r元")
            else:
                # FIX: Correctly reference the global dictionary
                friendly_error = ERROR_TRANSLATION.get(status, '未知错误')
                src.reply(f"§c查询失败: {friendly_error}§r\n详情: {data}")
                server.logger.warning(f"玩家 {src.name} 的手动查询失败 ({status}): {data}")
        
        do_check()

    # 处理原有的无参数指令: !!charge check
    def manual_server_query(src: CommandSource):
        src.reply("正在查询服务器默认电费，请稍候...")

        @new_thread('ManualDefaultCheck')
        def do_check():
            status, data = get_balance_from_api('44', '1207')
            if status == 'success':
                src.reply(f"§a查询成功！§r\n服务器默认房间当前余额：§6{data}§r元")
                try:
                    if float(data) < 15:
                        src.reply("§c警告：服务器电费余额已低于15元！")
                except (ValueError, TypeError):
                    pass
            else:
                # FIX: Correctly reference the global dictionary
                friendly_error = ERROR_TRANSLATION.get(status, '未知错误')
                src.reply(f"§c查询失败: {friendly_error}§r\n详情: {data}")
                server.logger.warning(f"玩家 {src.name} 的默认查询失败 ({status}): {data}")
        
        do_check()

    # 注册指令树
    server.register_command(
        Literal('!!charge').
        then(
            Literal('check').
            then(
                Text('building').
                then(
                    Text('room').runs(manual_user_query)
                )
            ).
            runs(manual_server_query)
        )
    )

    # 启动每小时自动检查的线程
    hourly_check_worker(server)