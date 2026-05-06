function handle(event) {
  return {
    action: "log",
    level: "info",
    message: "Sysmonitord 启动脚本执行成功，主机：" + event.hostname,
    allow_default: true,
    data: {
      event_id: event.event_id
    }
  };
}