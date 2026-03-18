---
name: feishu-channel-create
description: 自动创建飞书应用并配置本系统，形成可直接使用的飞书渠道
allowed-tools: browser(run_task) version(restart) edit_file
---

feishu channel create 是基于浏览器自动化创建飞书应用，并自动设置好所有本应用所需渠道权限、本应用配置信息等的技能。它通过读取本技能目录下assets资产中的对应json脚本，脚本字符串作为参数传递给browser工具的run_task动作，来完成自动化处理。

它通过以下步骤完成自动化处理：
1. 确认是否需要创建飞书应用
2. 确认应用名称
3. 通过前置脚本处理飞书应用的创建
4. 获取对应的信息更新配置
5. 重载配置
6. 通过后置脚本处理飞书应用的下一步配置

## 确认是否需要创建飞书应用
在开始创建飞书应用之前，需要先确认是否需要创建。如果用户确认需要创建，才会继续后续的处理。如果用户确认不需要创建，直接退出技能。

## 确认应用名称
与用户确认好新建的飞书应用名称，默认”望舒“，提醒用户不能有重名应用。

提醒用户，在首次使用时，需要用户在本系统内置浏览器中扫码或通过用户名密码等方式手动登录飞书，提前申请好飞书账号

## 通过前置脚本处理飞书应用的创建
### 1. 从assets资产中确认create.json脚本文件存在，作为前置处理脚本
前置处理脚本是一个json格式保存的文件，包含了自动化操作的所有信息，在执行前务必确保文件存在，并得到确切的文件位置。

### 2. 调用browser工具
调用browser工具时，需要传入参数，其中：
- action: run_task
- script_file: 前置处理脚本的文件绝对路径
- variables: 
  - app_name: 飞书应用名称，由用户指定，默认为”望舒“

传入参数示例：
```json
{
  "action": "run_task",
  "script_file": "/path/to/create.json",
  "variables": {
    "app_name": "望舒"
  }
}
```

### 3. 处理browser工具返回结果
browser工具会返回一个json格式的字符串，包含了执行结果。根据返回结果，判断是否创建成功。如果成功，继续后续处理；如果失败，提示用户检查并重新操作。

返回结果示例：
```json
{
  "success": true,
  "duration": 12345, // 耗时
  "error": "错误信息",
  "data": {
    "app_id": "创建后得到的app_id",
    "app_secret": "创建后得到的app_secret",
  }
}
```

## 获取对应的信息更新配置
### 1. 从返回结果中提取app_id和app_secret
根据browser工具返回的json字符串，提取出创建成功后的app_id和app_secret。

### 2. 更新本应用配置
将提取到的app_id和app_secret，调用configtool工具更新到本应用的配置文件中。
更新时，需要注意，在 "channels"节点下增加一个新的子节点，节点名称不要重复，如"wangshu_feishu"，节点内容为：
```json
{
  "action": "add",
  "section": "channels",
  "name": "wangshu_feishu",
  "data": "{\"type\": \"feishu\", \"enabled\": true, \"agent\": \"default\", \"app_id\": \"创建后得到的app_id\", \"app_secret\": \"创建后得到的app_secret\"}"
}
```

## 重载配置
调用configtool工具，传入参数
- action: reload

## 通过后置脚本处理飞书应用的下一步配置
### 1. 从assets资产中确认post.json脚本文件存在，作为后置处理脚本
后置处理脚本是一个json格式保存的文件，包含了自动化操作的所有信息，在执行前务必确保文件存在，并得到确切的文件位置。

### 2. 调用browser工具
调用browser工具时，需要传入参数，其中：
- action: run_task
- script_file: 后置处理脚本的文件绝对路径
- variables: 
  - app_id: 前序得到的飞书应用app_id

## 完成所有工作
在调用成功后即完成所有工作，可以释放本技能了