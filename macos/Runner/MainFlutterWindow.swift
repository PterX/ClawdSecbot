import Cocoa
import FlutterMacOS
import ServiceManagement

class MainFlutterWindow: NSWindow {
  override func awakeFromNib() {
    // 1. 先创建并启动 FlutterEngine，确保引擎在插件注册前完全就绪
    let engine = FlutterEngine(name: "main", project: nil, allowHeadlessExecution: true)
    engine.run(withEntrypoint: nil)
    
    // 2. 使用已启动的引擎创建 FlutterViewController
    let flutterViewController = FlutterViewController(engine: engine, nibName: nil, bundle: nil)
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)
    self.backgroundColor = NSColor(calibratedRed: 15.0 / 255.0, green: 15.0 / 255.0, blue: 35.0 / 255.0, alpha: 1.0)
    self.isOpaque = true
    
    // 3. 注册所有生成的插件（此时引擎已启动运行）
    // 注意: 多窗口插件的回调设置在 AppDelegate.applicationDidFinishLaunching
    RegisterGeneratedPlugins(registry: flutterViewController)
    
    // 4. 注册自定义 method channel
    let channel = FlutterMethodChannel(
      name: "launch_at_startup",
      binaryMessenger: engine.binaryMessenger
    )
    channel.setMethodCallHandler { [weak self] (call, result) in
      switch call.method {
      case "launchAtStartupIsEnabled":
        result(self?.isLaunchAtLoginEnabled() ?? false)
      case "launchAtStartupSetEnabled":
        if let args = call.arguments as? [String: Any],
           let enabled = args["setEnabledValue"] as? Bool {
          self?.setLaunchAtLogin(enabled: enabled)
        }
        result(nil)
      default:
        result(FlutterMethodNotImplemented)
      }
    }

    super.awakeFromNib()
    
    // 监听窗口样式变化,持续隐藏按钮
    DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) { [weak self] in
      self?.hideTrafficLights()
    }
    
    // 添加观察者,当styleMask改变时重新隐藏按钮
    self.addObserver(self, forKeyPath: "styleMask", options: [.new], context: nil)
  }
  
  // MARK: - Launch at Login (LaunchAgent based)
  
  private func launchAgentPlistPath() -> String {
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    return "\(home)/Library/LaunchAgents/com.bot.secnova.clawdsecbot.plist"
  }
  
  private func isLaunchAtLoginEnabled() -> Bool {
    return FileManager.default.fileExists(atPath: launchAgentPlistPath())
  }
  
  private func setLaunchAtLogin(enabled: Bool) {
    let plistPath = launchAgentPlistPath()
    
    if enabled {
      // Create LaunchAgent plist
      let appPath = Bundle.main.bundlePath
      let plistContent: [String: Any] = [
        "Label": "com.bot.secnova.clawdsecbot",
        "ProgramArguments": ["\(appPath)/Contents/MacOS/ClawdSecbot"],
        "RunAtLoad": true,
        "KeepAlive": false
      ]
      
      // Ensure LaunchAgents directory exists
      let launchAgentsDir = (plistPath as NSString).deletingLastPathComponent
      try? FileManager.default.createDirectory(atPath: launchAgentsDir, withIntermediateDirectories: true)
      
      // Write plist
      let plistData = try? PropertyListSerialization.data(fromPropertyList: plistContent, format: .xml, options: 0)
      try? plistData?.write(to: URL(fileURLWithPath: plistPath))
    } else {
      // Remove LaunchAgent plist
      try? FileManager.default.removeItem(atPath: plistPath)
    }
  }
  
  override func observeValue(forKeyPath keyPath: String?, of object: Any?, change: [NSKeyValueChangeKey : Any]?, context: UnsafeMutableRawPointer?) {
    if keyPath == "styleMask" {
      DispatchQueue.main.async { [weak self] in
        self?.hideTrafficLights()
      }
    }
  }
  
  override func becomeKey() {
    super.becomeKey()
    hideTrafficLights()
  }
  
  override func becomeMain() {
    super.becomeMain()
    hideTrafficLights()
  }
  
  private func hideTrafficLights() {
    standardWindowButton(.closeButton)?.isHidden = true
    standardWindowButton(.miniaturizeButton)?.isHidden = true
    standardWindowButton(.zoomButton)?.isHidden = true
  }
  
  deinit {
    self.removeObserver(self, forKeyPath: "styleMask")
  }
}
