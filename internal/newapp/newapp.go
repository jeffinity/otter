package newapp

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeffinity/otter/pkg/logx"
)

const (
	DefaultLayoutRepo = "https://github.com/jeffinity/app-layout.git"
	templateModule    = "github.com/jeffinity/app-layout"
	templateSubDir    = "app/app_layout"
)

type Options struct {
	LayoutSource  string
	OutputDir     string
	MonoRepo      bool
	SkipPostTasks bool
}

type routeKind int

const (
	routeSingleRepo routeKind = iota
	routeMonoCreateRepo
	routeMonoCreateApp
)

type copyKind int

const (
	copyFromLayoutRoot copyKind = iota
	copyFromTemplateDir
)

type creationRoute struct {
	kind                routeKind
	label               string
	targetDir           string
	modulePath          string
	copyStrategy        copyKind
	flattenAppLayout    bool
	renameTemplateDir   bool
	rewriteSingleImport bool
	runPostTasks        bool
	mustNotExist        bool
}

func Run(modulePath, appName string, opts Options) error {
	start := time.Now()
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "."
	}
	layoutSource := opts.LayoutSource
	if strings.TrimSpace(layoutSource) == "" {
		layoutSource = DefaultLayoutRepo
	}

	absOutputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return fmt.Errorf("解析输出目录失败: %w", err)
	}

	route, err := resolveCreationRoute(absOutputDir, strings.TrimSpace(modulePath), appName, opts)
	if err != nil {
		logError("路由解析失败: app=%s output=%s err=%v", appName, absOutputDir, err)
		return err
	}
	logStep("开始创建应用: route=%s module=%s app=%s output=%s", route.label, route.modulePath, appName, absOutputDir)

	layoutRoot, cleanup, err := fetchLayout(layoutSource)
	if err != nil {
		logError("拉取 layout 失败: route=%s app=%s err=%v", route.label, appName, err)
		return err
	}
	defer cleanup()

	sourceDir := filepath.Join(layoutRoot, templateSubDir)
	if !isDir(sourceDir) {
		err := fmt.Errorf("模板目录不存在: %s", sourceDir)
		logError("模板目录检查失败: route=%s dir=%s err=%v", route.label, sourceDir, err)
		return err
	}

	if err := executeRoute(route, appName, layoutRoot, sourceDir); err != nil {
		logError("应用创建失败: route=%s app=%s err=%v", route.label, appName, err)
		return err
	}
	logStep("应用创建完成: app=%s route=%s elapsed=%s", appName, route.label, time.Since(start).Round(time.Millisecond))
	return nil
}

func resolveCreationRoute(outputDir, modulePath, appName string, opts Options) (creationRoute, error) {
	if !opts.MonoRepo {
		logStep("路由选择: 单仓模式")
		return creationRoute{
			kind:                routeSingleRepo,
			label:               "单仓模式",
			targetDir:           filepath.Join(outputDir, appName),
			modulePath:          modulePath,
			copyStrategy:        copyFromLayoutRoot,
			flattenAppLayout:    true,
			rewriteSingleImport: true,
			runPostTasks:        !opts.SkipPostTasks,
		}, nil
	}

	if modulePath != "" {
		logStep("路由选择: 大仓模式(指定包名/建仓)")
		return creationRoute{
			kind:              routeMonoCreateRepo,
			label:             "大仓模式(指定包名/建仓)",
			targetDir:         filepath.Join(outputDir, appName),
			modulePath:        modulePath,
			copyStrategy:      copyFromLayoutRoot,
			flattenAppLayout:  false,
			renameTemplateDir: true,
			mustNotExist:      true,
		}, nil
	}

	appRoot := filepath.Join(outputDir, "app")
	if !isDir(appRoot) {
		return creationRoute{}, fmt.Errorf("大仓模式(仅应用名)要求输出目录下已存在 app 子目录: %s", appRoot)
	}
	logStep("路由选择: 大仓模式(仅应用名/建 app)")
	autoModulePath, err := detectModulePath(outputDir)
	if err != nil {
		return creationRoute{}, err
	}
	logStep("大仓模式(仅应用名): 自动识别模块路径=%s", autoModulePath)
	return creationRoute{
		kind:             routeMonoCreateApp,
		label:            "大仓模式(仅应用名/建 app)",
		targetDir:        filepath.Join(appRoot, appName),
		modulePath:       autoModulePath,
		copyStrategy:     copyFromTemplateDir,
		flattenAppLayout: false,
		mustNotExist:     true,
	}, nil
}

func executeRoute(route creationRoute, appName, layoutRoot, sourceDir string) error {
	logStep("%s: 目标目录=%s", route.label, route.targetDir)
	if route.mustNotExist && pathExists(route.targetDir) {
		return fmt.Errorf("目录已存在: %s", route.targetDir)
	}
	if err := ensureEmptyDir(route.targetDir); err != nil {
		return err
	}

	copySource := routeCopySource(route, layoutRoot, sourceDir)
	logStep("%s: 复制模板到目标目录", route.label)
	if err := copyDir(copySource, route.targetDir); err != nil {
		return fmt.Errorf("复制模板到输出目录失败: %w", err)
	}

	if err := applyRouteDirTransforms(route, route.targetDir, appName); err != nil {
		return err
	}

	logStep("%s: 执行模板替换", route.label)
	if err := replaceTemplateTokens(route.targetDir, route.modulePath, appName); err != nil {
		return err
	}

	if route.rewriteSingleImport {
		oldImportPrefix := fmt.Sprintf("%s/app/%s/", route.modulePath, appName)
		if err := replaceInFiles(route.targetDir, oldImportPrefix, route.modulePath+"/", onlyGoFiles); err != nil {
			return err
		}
	}

	if route.runPostTasks {
		if err := runSingleRepoPostTasks(route.targetDir); err != nil {
			return err
		}
	} else {
		logStep("%s: 跳过 conf/wire 后置任务", route.label)
	}
	return nil
}

func routeCopySource(route creationRoute, layoutRoot, sourceDir string) string {
	if route.copyStrategy == copyFromTemplateDir {
		return sourceDir
	}
	return layoutRoot
}

func applyRouteDirTransforms(route creationRoute, targetDir, appName string) error {
	if route.flattenAppLayout {
		if err := flattenAppLayoutToRoot(targetDir); err != nil {
			return err
		}
	}
	if route.renameTemplateDir {
		if err := renameTemplateAppDir(targetDir, appName); err != nil {
			return err
		}
	}
	return nil
}

func createStandaloneRepoWithLayout(outputDir, modulePath, appName, layoutRoot string) error {
	route := creationRoute{
		kind:              routeMonoCreateRepo,
		label:             "大仓模式(指定包名/建仓)",
		targetDir:         outputDir,
		modulePath:        modulePath,
		copyStrategy:      copyFromLayoutRoot,
		flattenAppLayout:  false,
		renameTemplateDir: true,
		mustNotExist:      true,
	}
	return executeRoute(route, appName, layoutRoot, filepath.Join(layoutRoot, templateSubDir))
}

func createInSingleRepo(outputDir, modulePath, appName, layoutRoot string, runPostTasks bool) error {
	route := creationRoute{
		kind:                routeSingleRepo,
		label:               "单仓模式",
		targetDir:           outputDir,
		modulePath:          modulePath,
		copyStrategy:        copyFromLayoutRoot,
		flattenAppLayout:    true,
		rewriteSingleImport: true,
		runPostTasks:        runPostTasks,
	}
	return executeRoute(route, appName, layoutRoot, filepath.Join(layoutRoot, templateSubDir))
}

func createInMonoRepo(outputDir, modulePath, appName, sourceDir string) error {
	appRoot := filepath.Join(outputDir, "app")
	if !isDir(appRoot) {
		return fmt.Errorf("大仓模式要求输出目录下已存在 app 子目录: %s", appRoot)
	}
	route := creationRoute{
		kind:             routeMonoCreateApp,
		label:            "大仓模式(仅应用名/建 app)",
		targetDir:        filepath.Join(appRoot, appName),
		modulePath:       modulePath,
		copyStrategy:     copyFromTemplateDir,
		flattenAppLayout: false,
		mustNotExist:     true,
	}
	return executeRoute(route, appName, filepath.Dir(filepath.Dir(sourceDir)), sourceDir)
}

func flattenAppLayoutToRoot(outputDir string) error {
	appDir := filepath.Join(outputDir, "app")
	appLayoutDir := filepath.Join(appDir, "app_layout")
	if !isDir(appLayoutDir) {
		return fmt.Errorf("单仓模式需要模板中存在目录: %s", appLayoutDir)
	}

	if err := moveDirContents(appLayoutDir, outputDir); err != nil {
		return fmt.Errorf("搬移 app/app_layout 到项目根目录失败: %w", err)
	}
	logStep("单仓模式: 已将 app/app_layout 内容搬移到项目根目录")
	if err := os.RemoveAll(appDir); err != nil {
		return fmt.Errorf("删除 app 目录失败: %w", err)
	}
	return nil
}

func renameTemplateAppDir(outputDir, appName string) error {
	appDir := filepath.Join(outputDir, "app")
	from := filepath.Join(appDir, "app_layout")
	to := filepath.Join(appDir, appName)
	if !isDir(from) {
		return fmt.Errorf("大仓建仓模式需要模板中存在目录: %s", from)
	}
	if pathExists(to) {
		return fmt.Errorf("目标路径已存在: %s", to)
	}
	if err := os.Rename(from, to); err != nil {
		return fmt.Errorf("重命名应用目录失败: %w", err)
	}
	logStep("大仓建仓模式: 已将 app/app_layout 重命名为 app/%s", appName)
	return nil
}

func replaceTemplateTokens(rootDir, modulePath, appName string) error {
	safeName := strings.ReplaceAll(appName, "-", "_")

	if err := replaceInFiles(rootDir, "app_layout", safeName, onlyProtoFiles); err != nil {
		return err
	}
	if err := replaceInFiles(rootDir, "app_layout", appName, nil); err != nil {
		return err
	}
	if err := replaceInFiles(rootDir, templateModule, modulePath, onlyGoFiles); err != nil {
		return err
	}

	goMod := filepath.Join(rootDir, "go.mod")
	if pathExists(goMod) {
		if err := replaceInFile(goMod, templateModule, modulePath); err != nil {
			return err
		}
	}

	golangCILint := filepath.Join(rootDir, ".golangci.yml")
	if pathExists(golangCILint) {
		if err := replaceInFile(golangCILint, templateModule, modulePath); err != nil {
			return err
		}
	}
	return nil
}

func fetchLayout(layoutSource string) (layoutRoot string, cleanup func(), err error) {
	tempDir, err := os.MkdirTemp("", "newapp-layout-*")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}
	cleanup = func() {
		_ = os.RemoveAll(tempDir)
	}

	layoutRoot = filepath.Join(tempDir, "layout")
	if isDir(layoutSource) {
		logStep("使用本地 layout 源: %s", layoutSource)
		if err := copyDir(layoutSource, layoutRoot); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("复制本地 layout 源失败: %w", err)
		}
		logStep("本地 layout 复制完成: %s", layoutRoot)
		return layoutRoot, cleanup, nil
	}

	if _, lookErr := exec.LookPath("git"); lookErr != nil {
		cleanup()
		return "", nil, fmt.Errorf("未找到 git，请先安装 git 或用 -r 指定本地目录")
	}

	logStep("开始 git clone layout: source=%s target=%s", layoutSource, layoutRoot)
	cmd := exec.Command(
		"git", "-c", "advice.detachedHead=false",
		"clone", "--depth=1", "--recurse-submodules", "--shallow-submodules",
		layoutSource, layoutRoot,
	)
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		cleanup()
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", nil, fmt.Errorf("拉取 layout 失败: %w", runErr)
		}
		return "", nil, fmt.Errorf("拉取 layout 失败: %w: %s", runErr, msg)
	}
	logStep("git clone 完成: %s", layoutRoot)
	return layoutRoot, cleanup, nil
}

func logStep(format string, args ...any) {
	logx.Infof(format, args...)
}

func logError(format string, args ...any) {
	logx.Errorf(format, args...)
}

func runSingleRepoPostTasks(projectDir string) error {
	logStep("单仓模式: 执行 task conf")
	if err := runTask(projectDir, "conf"); err != nil {
		return err
	}

	logStep("单仓模式: 执行 task wire -- .")
	if err := runTask(projectDir, "wire", "--", "."); err != nil {
		return err
	}
	logStep("单仓模式: task conf/wire 执行完成")
	return nil
}

func runTask(dir string, args ...string) error {
	if _, err := exec.LookPath("task"); err != nil {
		return fmt.Errorf("未找到 task 命令，请先安装 go-task: %w", err)
	}

	cmd := exec.Command("task", args...)
	cmd.Dir = dir
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("执行 task %s 失败: %w", strings.Join(args, " "), runErr)
		}
		return fmt.Errorf("执行 task %s 失败: %w\n%s", strings.Join(args, " "), runErr, msg)
	}
	return nil
}

func detectModulePath(workDir string) (string, error) {
	cmd := exec.Command("go", "list", "-m")
	cmd.Dir = workDir
	out, runErr := cmd.CombinedOutput()
	if runErr != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", fmt.Errorf("自动识别模块路径失败: %w", runErr)
		}
		return "", fmt.Errorf("自动识别模块路径失败: %w: %s", runErr, msg)
	}
	modulePath := strings.TrimSpace(string(out))
	if modulePath == "" {
		return "", fmt.Errorf("自动识别模块路径失败: go list -m 返回为空")
	}
	return modulePath, nil
}

func ensureEmptyDir(path string) error {
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		return os.MkdirAll(path, 0o755)
	case err != nil:
		return fmt.Errorf("检查输出目录失败: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("输出路径不是目录: %s", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("读取输出目录失败: %w", err)
	}
	if len(entries) > 0 {
		return fmt.Errorf("单仓模式要求输出目录为空: %s", path)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.Name() == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			// Skip submodule metadata files such as app/.../.git.
			return nil
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode())
		}

		if d.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func moveDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		if pathExists(dstPath) {
			return fmt.Errorf("目标路径已存在: %s", dstPath)
		}
		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

func replaceInFiles(rootDir, oldValue, newValue string, matcher func(string) bool) error {
	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if matcher != nil && !matcher(path) {
			return nil
		}
		return replaceInFile(path, oldValue, newValue)
	})
}

func replaceInFile(path, oldValue, newValue string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if isBinary(data) || !bytes.Contains(data, []byte(oldValue)) {
		return nil
	}

	newData := bytes.ReplaceAll(data, []byte(oldValue), []byte(newValue))
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, newData, info.Mode()); err != nil {
		return err
	}
	return nil
}

func isBinary(data []byte) bool {
	return bytes.IndexByte(data, 0) >= 0
}

func onlyProtoFiles(path string) bool {
	return filepath.Ext(path) == ".proto"
}

func onlyGoFiles(path string) bool {
	return filepath.Ext(path) == ".go"
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
