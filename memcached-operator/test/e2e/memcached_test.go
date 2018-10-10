package e2e
// Once MainEntry sets up the framework, it runs the remainder of the tests. First, make sure to import testing, 
// the operator-sdk test  framework (pkg/test) as well as your operator's libraries:
import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	cachev1alpha1 "github.com/example-inc/memcached-operator/pkg/apis/cache/v1alpha1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	retryInterval = time.Second * 5
	timeout       = time.Second * 30
)

func TestMemcached(t *testing.T) {

// ***************1**********************
// The next step is to register your operator's scheme with the framework's dynamic client. 	
// To do this, pass the CRD's AddToScheme function and its List type object to the framework's AddToFrameworkScheme function.
// For our example memcached-operator, it looks like this:

	memcachedList := &cachev1alpha1.MemcachedList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Memcached",
			APIVersion: "cache.example.com/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(cachev1alpha1.AddToScheme, memcachedList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
	// run subtests
	t.Run("memcached-group", func(t *testing.T) {
		t.Run("Cluster", MemcachedCluster)
		t.Run("Cluster2", MemcachedCluster)
	})
}

func memcachedScaleTest(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		return fmt.Errorf("could not get namespace: %v", err)
	}
// ***************2**********************
// Now that the operator is ready, we can create a custom resource. 
// Since the controller-runtime's dynamic client uses go contexts, make sure to import the go context library. 
// In this example, we imported it as goctx:

        // create memcached custom resource
	exampleMemcached := &cachev1alpha1.Memcached{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Memcached",
			APIVersion: "cache.example.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-memcached",
			Namespace: namespace,
		},
		Spec: cachev1alpha1.MemcachedSpec{
			Size: 3,
		},
	}
	err = f.DynamicClient.Create(goctx.TODO(), exampleMemcached)
	if err != nil {
		return err
	}
	ctx.AddFinalizerFn(func() error {
		return f.DynamicClient.Delete(goctx.TODO(), exampleMemcached)
	})

	// wait for example-memcached to reach 3 replicas
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached", 3, retryInterval, timeout)
	if err != nil {
		return err
	}

	err = f.DynamicClient.Get(goctx.TODO(), types.NamespacedName{Name: "example-memcached", Namespace: namespace}, exampleMemcached)
	if err != nil {
		return err
	}
	exampleMemcached.Spec.Size = 4
	err = f.DynamicClient.Update(goctx.TODO(), exampleMemcached)
	if err != nil {
		return err
	}

	// wait for example-memcached to reach 4 replicas
	return e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "example-memcached", 4, retryInterval, timeout)
}

func MemcachedCluster(t *testing.T) {
	t.Parallel()

// ***************3**********************
// The next step is to create a TestCtx for the current test and defer its cleanup function:

	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup(t)
	err := ctx.InitializeClusterResources()
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global
	// wait for memcached-operator to be ready
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "memcached-operator", 1, retryInterval, timeout)
	if err != nil {
		t.Fatal(err)
	}

	if err = memcachedScaleTest(t, f, ctx); err != nil {
		t.Fatal(err)
	}
}
