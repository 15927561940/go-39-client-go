package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"k8s.io/client-go/util/retry"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"path/filepath"

	appsv1 "k8s.io/api/apps/v1" //第三方包定义别名
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/homedir"
)

func main() {
	//定义要用到的kubeconfig,ns,deployment的名字
	var kubeconfig, namespace, name *string //操作集群需要的基本的资源

	//获取kubeconfig文件,分不同环境是否存在/root家目录
	if home := homedir.HomeDir(); home != "" {
		//home目录存在的kubeconfig的默认位置
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "optional可选的kubeconfig目录")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "需要指定kubeconfig目录的位置给我用")
	}

	//定义ns
	namespace = flag.String("namespace请输入", "default", "可选，不选则默认default")

	//定义deployment的名字
	name = flag.String("name-deployment的名字", "demo-deployment", "可选，不选默认为demo-deployment")

	//通过flag接受上面定义的参数
	flag.Parse()

	//拼接kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	//生成K8S的总对象，可以用来操作具体的对象，deployment，ds等
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	//指定版本和对象
	dpClient := clientset.AppsV1().Deployments(*namespace)
	if err != nil {
		panic(err)
	}

	//定义dp的yaml资源
	//其实就类似创建dp需要的资源
	deploy := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: *name,
		},

		//传入spec下的内容
		Spec: appsv1.DeploymentSpec{
			Replicas: intToPtr(2), //传入的要是指针,定义一个方法转成指针
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
				MatchExpressions: nil,
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Volumes:        nil,
					InitContainers: nil,
					Containers: []apiv1.Container{
						{
							Name:  *name,
							Image: "nginx:1.19.8",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
					EphemeralContainers:           nil,
					RestartPolicy:                 "",
					TerminationGracePeriodSeconds: nil,
					ActiveDeadlineSeconds:         nil,
					DNSPolicy:                     "",
					NodeSelector:                  nil,
					ServiceAccountName:            "",
					DeprecatedServiceAccount:      "",
					AutomountServiceAccountToken:  nil,
					NodeName:                      "",
					HostNetwork:                   false,
					HostPID:                       false,
					HostIPC:                       false,
					ShareProcessNamespace:         nil,
					SecurityContext:               nil,
					ImagePullSecrets:              nil,
					Hostname:                      "",
					Subdomain:                     "",
					Affinity:                      nil,
					SchedulerName:                 "",
					Tolerations:                   nil,
					HostAliases:                   nil,
					PriorityClassName:             "",
					Priority:                      nil,
					DNSConfig:                     nil,
					ReadinessGates:                nil,
					RuntimeClassName:              nil,
					EnableServiceLinks:            nil,
					PreemptionPolicy:              nil,
					Overhead:                      nil,
					TopologySpreadConstraints:     nil,
					SetHostnameAsFQDN:             nil,
					OS:                            nil,
					HostUsers:                     nil,
					SchedulingGates:               nil,
					ResourceClaims:                nil,
				},
			},
		},
		Status: appsv1.DeploymentStatus{},
	}

	//开始创建deployment
	fmt.Println("//开始创建deployment")

	result, err := dpClient.Create(context.TODO(), deploy, metav1.CreateOptions{
		TypeMeta:        metav1.TypeMeta{},
		DryRun:          nil,
		FieldManager:    "",
		FieldValidation: "",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("创建deployment成功%s在ns%s中", result.GetObjectMeta().GetName(), result.GetObjectMeta().GetNamespace())

	//等待键盘enter后继续
	prompt()

	//更新deployment
	fmt.Println("开始更新镜像")

	//重试几次
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		result, getErr := dpClient.Get(context.TODO(), *name, metav1.GetOptions{
			TypeMeta:        metav1.TypeMeta{},
			ResourceVersion: "",
		})
		if getErr != nil {
			panic(fmt.Errorf("获取最新版本的deployment失败：%v", getErr))
		}

		//无报错则该数量为1
		result.Spec.Replicas = intToPtr(1)
		result.Spec.Template.Spec.Containers[0].Image = "nginx:1.25.5"
		_, updateErr := dpClient.Update(context.TODO(), result, metav1.UpdateOptions{})
		return updateErr

	})
	if retryErr != nil {
		panic(fmt.Errorf("update  failed:%v", retryErr))
	}
	fmt.Println("deployment更新正常")

	prompt()

	//列出deployment资源
	fmt.Printf("列出deployment资源%s", *namespace)

	list, err := dpClient.List(context.TODO(), metav1.ListOptions{
		TypeMeta:             metav1.TypeMeta{},
		LabelSelector:        "",
		FieldSelector:        "",
		Watch:                false,
		AllowWatchBookmarks:  false,
		ResourceVersion:      "",
		ResourceVersionMatch: "",
		TimeoutSeconds:       nil,
		Limit:                0,
		Continue:             "",
		SendInitialEvents:    nil,
	})
	if err != nil {
		panic(err)
	}

	//列出
	for _, item := range list.Items {
		fmt.Printf("%s (%d  replicas)\n", item.Name, *&item.Spec.Replicas)
	}
	prompt()

	//删除deployment
	fmt.Println("开始删除deployment")
	deletePolicy := metav1.DeletePropagationForeground
	if err := dpClient.Delete(context.TODO(), *name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		panic(err)
	}

	fmt.Println("删除成功")
}

func intToPtr(i int32) *int32 {
	return &i
}

func prompt() {
	fmt.Println("按enter回车键继续")
	//无回车则阻塞
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		break
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

}
