github# deploy-scaler — Kubernetes Operator (Kubebuilder)

## 三条初始化命令详解

### 1. `kubebuilder create api --kind Scaler --version v1vita --group api`

**作用**：在项目中创建一个完整的 Kubernetes Operator 骨架，包含 CRD 定义 + Controller 控制器。

**参数拆解**：
| 参数 | 值 | 含义 |
|------|-----|------|
| `--kind Scaler` | Scaler | 自定义资源的**类型名称**（Kind），生成 `Scaler` 和 `ScalerList` 两个 Go 类型 |
| `--version v1vita` | v1vita | API **版本号**，对应 `apiVersion: api.scaler.com/v1vita` |
| `--group api` | api | API **组名**，与 `PROJECT` 中的 `domain: scaler.com` 组合成完整组名 `api.scaler.com` |

**关键标志**（默认启用）：
- `--controller=true`：生成 Controller 骨架（`internal/controller/scaler_controller.go`）
- `--resource=true`：生成 CRD 类型定义（`api/v1vita/scaler_types.go`）

**执行后生成的核心文件**：

```
api/v1vita/
├── scaler_types.go          # CRD 的 Spec/Status Go 类型定义
├── groupversion_info.go     # GroupVersion 注册（api.scaler.com/v1vita）
└── zz_generated.deepcopy.go # controller-gen 自动生成的 DeepCopy 方法

internal/controller/
├── scaler_controller.go     # Reconcile 循环逻辑（控制器核心）
├── scaler_controller_test.go # 控制器单元测试
└── suite_test.go            # Ginkgo 测试套件配置
```

> **注意**：你输入的两条命令（第 1 条 `--king` 是拼写错误，第 2 条才是正确的），实际生效的是第 2 条。

---

### 2. （同上，正确命令）

第 1 条命令 `--king` 是 `--kind` 的拼写错误，Kubebuilder 无法识别，会报错或忽略。第 2 条才是正确执行的命令。

---

### 3. `make manifests`

**作用**：根据 Go 代码中的 `// +kubebuilder` 注释标记（markers），生成 Kubernetes YAML 清单文件。

**生成的文件**：

| 文件 | 内容 |
|------|------|
| `config/crd/bases/api.scaler.com_scalers.yaml` | CRD 定义（Kubernetes 自定义资源类型） |
| `config/rbac/role.yaml` | RBAC 权限规则（Controller 对 Scaler 资源的 CRUD 权限） |
| `config/rbac/scaler_*_role.yaml` | 用户角色（admin/editor/viewer） |

**markers 驱动机制**：

代码中的注释标记是生成的"源码"，例如：

```go
// scaler_types.go
// +kubebuilder:object:root=true        → 生成 runtime.Object 接口实现
// +kubebuilder:subresource:status      → 生成 /status 子资源
```

```go
// scaler_controller.go
// +kubebuilder:rbac:groups=api.scaler.com,resources=scalers,verbs=get;list;watch;create;update;patch;delete
// → 生成对应的 RBAC ClusterRole 规则
```

**底层工具链**：
```
controller-gen  →  从 markers 生成 CRD YAML + DeepCopy Go 代码
kustomize       →  组装最终的 manifests
```

**何时需要重新运行**：
- 修改了 `*_types.go` 中的字段或 markers
- 修改了 Controller 中的 RBAC markers
- 每次 `make generate` 后建议跟随 `make manifests`

---

## 完整目录结构

```
deploy-scaler/
│
├── PROJECT                          # Kubebuilder 项目元数据（自动生成，勿手动编辑）
├── Makefile                         # 构建/测试/部署命令入口
├── go.mod / go.sum                  # Go 模块依赖
├── Dockerfile                       # Operator 容器镜像构建文件
├── .dockerignore                    # Docker 构建排除规则
├── .golangci.yml                    # GolangCI-Lint 配置
├── README.md                        # 本文档
│
├── cmd/
│   └── main.go                      # Operator 入口（Manager 启动、Controller 注册、Webhook 注册）
│
├── api/
│   └── v1vita/                      # API 版本目录
│       ├── scaler_types.go          # CRD 定义：ScalerSpec + ScalerStatus + Scaler + ScalerList
│       ├── groupversion_info.go     # GroupVersion 定义（api.scaler.com/v1vita）+ Scheme 注册
│       └── zz_generated.deepcopy.go # DeepCopy 方法（自动生成，勿手动编辑）
│
├── internal/
│   └── controller/
│       ├── scaler_controller.go     # Reconcile 循环核心逻辑
│       ├── scaler_controller_test.go # 控制器单元测试
│       └── suite_test.go            # envtest 测试环境配置
│
├── config/                          # Kustomize 配置（用于部署到 K8s 集群）
│   ├── crd/
│   │   ├── bases/
│   │   │   └── api.scaler.com_scalers.yaml   # CRD 定义（make manifests 生成）
│   │   ├── kustomization.yaml
│   │   └── kustomizeconfig.yaml
│   │
│   ├── rbac/
│   │   ├── role.yaml               # Controller 权限（make manifests 生成）
│   │   ├── role_binding.yaml       # Role 绑定到 ServiceAccount
│   │   ├── service_account.yaml    # Operator 运行的 ServiceAccount
│   │   ├── scaler_admin_role.yaml  # 管理员角色
│   │   ├── scaler_editor_role.yaml # 编辑者角色
│   │   ├── scaler_viewer_role.yaml # 只读角色
│   │   ├── leader_election_role.yaml         # Leader 选举权限
│   │   ├── leader_election_role_binding.yaml
│   │   ├── metrics_auth_role.yaml            # Metrics 访问权限
│   │   ├── metrics_reader_role.yaml
│   │   ├── metrics_auth_role_binding.yaml
│   │   └── kustomization.yaml
│   │
│   ├── manager/
│   │   ├── manager.yaml            # Operator Deployment 定义
│   │   └── kustomization.yaml
│   │
│   ├── samples/
│   │   ├── api_v1vita_scaler.yaml  # 示例 CR（kubectl apply 测试用）
│   │   └── kustomization.yaml
│   │
│   ├── default/                    # 默认部署配置（组合上述所有模块）
│   │   ├── kustomization.yaml
│   │   ├── manager_metrics_patch.yaml
│   │   ├── metrics_service.yaml
│   │   └── cert_metrics_manager_patch.yaml
│   │
│   ├── prometheus/                 # Prometheus 监控配置
│   │   ├── monitor.yaml
│   │   ├── monitor_tls_patch.yaml
│   │   └── kustomization.yaml
│   │
│   └── network-policy/             # 网络策略
│       ├── allow-metrics-traffic.yaml
│       └── kustomization.yaml
│
├── test/
│   └── e2e/                        # E2E 测试（需 Kind 集群）
│       └── system/
│           └── e2e_suite_test.go
│           └── e2e_test.go
│
├── hack/
│   └── boilerplate.go.txt          # 代码生成时的文件头模板
│
├── .github/workflows/              # GitHub Actions CI/CD
│   └── test.yml
│
├── .devcontainer/                  # VS Code Dev Container 配置
└── bin/                            # 本地工具二进制（controller-gen 等）
```

---

## 核心组件关系图

```
cmd/main.go（Manager 入口）
    │
    ├── 注册 Scheme（client-go 内置 + 自定义 Scaler CRD）
    │
    ├── 创建 Manager（连接 K8s API Server、启动 Informer）
    │
    └── 注册 ScalerReconciler
            │
            └── SetupWithManager
                    ├── For(&Scaler{})     ← 监听 Scaler CR 变化
                    └── Complete(r)        ← 绑定 Reconcile 循环

ScalerReconciler.Reconcile(ctx, req)
    │
    ├── r.Get(ctx, req.NamespacedName, &Scaler{})   ← 获取当前资源
    ├── 对比 Spec 与实际集群状态
    ├── 创建/更新/删除子资源（Deployment、Service 等）
    └── 更新 Status（r.Status().Update）
```

---

## 常用命令

```bash
# 重新生成 CRD/RBAC manifests（修改 types 或 markers 后必跑）
make manifests

# 重新生成 DeepCopy 方法（修改 types 后必跑）
make generate

# 运行单元测试（基于 envtest：真实 K8s API + etcd）
make test

# 本地运行 Operator（使用当前 kubeconfig）
make run

# 构建并部署到集群
export IMG=your-registry/deploy-scaler:v0.0.1
make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG

# 创建示例 CR
kubectl apply -f config/samples/api_v1vita_scaler.yaml

# 查看 Operator 日志
kubectl logs -n deploy-scaler-system deployment/deploy-scaler-controller-manager -c manager -f
```

---

## 下一步

1. **编辑 `api/v1vita/scaler_types.go`**：定义你需要的 Spec 字段（如 replicas、image、resources 等）
2. **运行 `make manifests generate`**：重新生成 CRD 和 DeepCopy
3. **编辑 `internal/controller/scaler_controller.go`**：在 `Reconcile` 函数中编写实际的业务逻辑
4. **测试**：`make run` 本地验证，或 `make deploy` 部署到测试集群
# deploy-scaler
# deploy-scaler
# deploy-scaler
