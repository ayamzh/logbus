# logbus
a simple logger dedicated to JSON output

## 背景

我们在开发环境下的log一般输出到本地文件，或者stdout（再由supervisor写文件）。 在线上环境将log统一打到log服务器（logserver）上，然后运维解析tag分发到 s3 / elasticsearch/ thinkingdata 等。 从服务器输出到logserver有多种方法：

- 直接输出到本地fluentd agent的socket中
- 若机器部署在k8s上，则可以输出到stdout，然后由fluentd-bit采集文件和分发
- 若机器部署在ec2上，则可以输出到stdout，然后由supervisor重定向到文件，运维对文件进行采集

## 基础用法

```go
func main() {
	// close logger before exit
	defer logbus.Close()

    // Init with conf 线上WithDev()要填false
	logbus.Init(logbus.NewConf(logbus.WithOutputStdout(true), logbus.WithDev(true)))

	// Print server info log, dd_meta_channel=logbus.SERVERLOG
	logbus.Info(zap.Int("int", 123))

	// 和上条等价
	logbus.Logger().Info(zap.Int("int", 123))

	// Print bi log, dd_meta_channel=bi
	logbus.Logger().L().Info("bi", zap.Int("money", 648))
}
```

logbus会将zap的MessageKey设定为dd_meta_channel，用于标记log的属性。 logbus.Debug() 或 logbus.Logger().Debug() 会使用默认的channel: logbus.SERVERLOG。 logbus.Logger().L().Debug()的第一个参数可以指定自定义的channel。 运维可以根据这个字段来建立分发规则。例如"dd_meta_channel":"server"打到elasticsearch，"dd_meta_channel":"thinkingdata"打到tga，"dd_meta_channel":"bigquery" 打到s3。

# 改变其它库的输出
sandwich >= 1.2 内部默认使用lobus.Logger()输出log. 推荐使用自定义logger，在代码中调用logbus.SetGlogLogger()即可改变sandwich的内部logger
sandwich 1.2.9 增加了一个 slog.DefaultDepth 的全局变量，可以调整框架内部的caller depth，默认为1. sandwich框架内部的depth == 业务传入的logger的depth + slog.DefaultDepth

设置环境变量 export sandwich_slog_disable=1 可以关闭sandwich的log

xconf使用logbus输出

```go
xconf.Parse(ccPtr,xconf.WithLogWarning(func(s string) { logbus.Logger().Wanging(zap.String("xconf", s))}))
```

## 进阶用法
# 一 多地输出
支持输出 stdout, 文件 和 fluentd socket ，可以同时输出到以上0-3个地方。通过logbus.Init()时的配置控制。

文件输出通过把 gopkg.in/natefinch/lumberjack.v2 包裹成zapcore.WriteSyncer来实现。支持file rotation。传入的tag作为文件名，不同的tag输出到不同的文件中，使得log可以分类查询。一条log可以同时输出到多个文件中（logid一样）。文件输出：

```json
{
   "log_level":"debug",
   "date":"2020-10-30T15:28:27.405+0800",
   "dd_meta_channel":"server",
   "int":123,
   "time":"2020-10-30T15:28:27.405+0800",
   "log_xid":"buds16rc1osginerdie0"
}
```

fluentd socket 通过实现一个zapcore.Core来获取zap的[]zapcore.Field，然后调用 github.com/fluent/fluent-logger-golang 按tag进行分发。由于fluentd的要求：record must be a Hash: String，只能将[]zapcore.Field拼装成map[string]interface{}，性能开销上不如zap的JSONEncoder。fluentd输出：

```json
{
   "tga":{
      "#ip":"10.0.0.3",
      "properties":{
         "player_name":"zhang wu",
         "level":9
      },
      "#account_id":"111",
      "#type":"user_setOnce",
      "#time":"2020-10-30 16:20:27.281"
   },
   "log_xid":"budspirc1oshonq07dvg"
}
```

# 二 logid和扩展
默认每条log会带一个logid，字段名log_xid，来区分log的唯一性。可以在配置Conf里关闭。 提供 AppendGlobalFields 方法，可以供业务方在每条log内添加自定义的字段。见进阶用法代码示例 23行。

# 三 FieldQueue
```go
q := logbus.NewQueue()
q.Push(zap.Int("i", 1))
q.Push(zap.Int("j", 2))
logbus.Debug(q.Retrieve()...)
```

# 四 ThinkingData
本库集成了部分thinkingdata的SDK, 提出了两个接口：

thinkingdata.User() - 构造用户属性型log的Data
thinkingdata.Track() - 构造事件数据型log的Data
thinkingdata.ExtractEncoder() - 提取出 # 开头或 字母开头的 field，抛弃其他类型的prefix的field
~~对Data增加了MarshalLogObject()接口，用户可以使用zap.Object(“key”, Data)来输出thinkingdata的log。 ~~ (fluentd 支持的不好 depracated)
对Data增加了MarshalAsJson()接口，将data转成[]byte。
用户可以使用logbus.Logger().PrintThingkingData()来输出thinkingdata的log。

```go
properties := map[string]interface{}{"#ip": "10.0.0.2", "player_name": "zhang si", "level": 8}
data, err := thinkingdata.User("111", "", thinkingdata.USER_SET_ONCE, properties)
// Check Error
Logger().PrintThingkingData(data)
```
也可以直接使用 logbus.Logger(logbus.THINKINGDATA).Track()
```go
err := logbus.Logger(logbus.THINKINGDATA).Track(zap.String(thinkingdata.ACCOUNT, "111"), zap.String(thinkingdata.TYPE, thinkingdata.USER_SET_ONCE),
	zap.String("player_name", "zhang liu"), zap.Int("level", 11))
// Check Error
```
# 五 BigQuery
bigquery.ExtractEncoder() 提取出 $ 开头或 字母开头的 field，抛弃其他类型的prefix的field, 返回表名和聚合好的column。用户可以使用logbus.Logger().PrintBigQuery()来输出log

也可以直接使用 logbus.Logger(logbus.BIGQUERY).Track()

有如下参数可以配置

ColumnPrefix : 默认$ 。 以 ColumnPrefix_ _开头的field认为是bigquery的列名
TableNameKey： 默认$tablename. 等于TableNameKey的field的值认为是bigquery的表名
ColumnProperties：默认data. 以字母开头的field合并到一列，列名为ColumnProperties
UseRecord： 默认false. false则将data这一列json marshal后作为string存储，适合输出格式经常变动的场景。true则作为bigquery的record存储（必须符合已定义的schema才能load成功），适合输出格式不变动的场景
UseRecord true:
```json
{
   "log_level":"info",
   "date":"2020-11-26T16:41:05.025+0800",
   "dd_meta_channel":"bigquery",
   "tags":[
      "bigquery"
   ],
   "$tablename":"oplog",
   "msg":"{\"user_id\":\"111\",\"optime\":\"2020-11-26T16:41:05.025+0800\",\"data\":{\"player_name\":\"zhang liu\",\"level\":11,\"bool\":true,\"strings\":[\"x\",\"y\"]}}",
   "log_xid":"buvmk8bc1osgj2aojgd0"
}
```
# 六 一log两吃
提供有接口Track(), 从fields里根据key的命名，自动提取TGA和BigQuery各自所需元素，然后分发到指定的tag中。 比如进阶用法代码示例的第18行将会输出两条不一样的log到两个dd_meta_channel：
```json
{
   "log_level":"info",
   "date":"2020-11-26T16:41:05.025+0800",
   "dd_meta_channel":"bigquery",
   "tags":[
      "bigquery",
      "thinkingdata"
   ],
   "$tablename":"oplog",
   "msg":"{\"user_id\":\"111\",\"optime\":\"2020-11-26T16:41:05.025+0800\",\"data\":\"{\\\"player_name\\\":\\\"zhang liu\\\",\\\"level\\\":11,\\\"bool\\\":true,\\\"strings\\\":[\\\"x\\\",\\\"y\\\"]}\"}",
   "log_xid":"buvmk8bc1osgj2aojgc0"
}
{
   "log_level":"info",
   "date":"2020-11-26T16:41:05.025+0800",
   "dd_meta_channel":"thinkingdata",
   "tags":[
      "bigquery",
      "thinkingdata"
   ],
   "msg":"{\"#account_id\":\"111\",\"#type\":\"user_setOnce\",\"#time\":\"2020-11-26 16:41:05.025\",\"#properties\":{\"bool\":true,\"level\":11,\"player_name\":\"zhang liu\",\"strings\":[\"x\",\"y\"]}}",
   "log_xid":"buvmk8bc1osgj2aojgcg"
}
```
# 七 增加默认字段
在程序中插入下面的代码，则每条log都会携带host_name, server_id，server_birth这些自定义字段。
```go
	logbus.AppendGlobalFields(zap.String("host_name", hostName))
	logbus.AppendGlobalFields(zap.String("server_id", uuid.New().String()))
	logbus.AppendGlobalFields(zap.Int64("server_birth", time2.Unix()))
```


# 配置说明
常用的配置都在logbus.NewConf() 里

