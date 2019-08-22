package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	flags "github.com/jessevdk/go-flags"
	"golang.org/x/xerrors"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// Variables inserted from go build.
var (
	Version   string
	GoVersion string
	Revision  string
)

type options struct {
	File       string `short:"f" long:"file" description:"path to kubectl config file you want to read. read from stdin if not specifyed"`
	Kubeconfig string `long:"kubeconfig"`
	Version    bool   `long:"version"`
}

func main() {
	if err := execute(); err != nil {
		log.Printf("error: %+v", err)
		os.Exit(1)
	}
	os.Exit(0)
}

var opts options

func execute() error {
	if _, err := flags.Parse(&opts); err != nil {
		if flags.WroteHelp(err) {
			return nil
		}
		return xerrors.Errorf("parsing flags: %w", err)
	}
	if opts.Version {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Revision: %s\n", Revision)
		fmt.Printf("GoVersion: %s\n", strings.TrimPrefix(GoVersion, "go version "))
		return nil
	}

	kubeconfig, err := getKubeconfigPath(opts.Kubeconfig)
	if err != nil {
		return xerrors.Errorf("getting kubeconfig path: %w", err)
	}

	bkup, err := backup(kubeconfig)
	if err != nil {
		return xerrors.Errorf("error on backup: %s: %w", kubeconfig, err)
	}
	var shouldRollback bool
	defer func() {
		if shouldRollback {
			if err := rollback(bkup, kubeconfig); err != nil {
				log.Printf("error on rollback: %+v", err)
			}
		}
	}()

	r := os.Stdin
	if opts.File != "" {
		f, err := os.Open(opts.File)
		if err != nil {
			return xerrors.Errorf("opening file: %s: %w", opts.File, err)
		}
		defer f.Close()
		r = f
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	cfg, err := clientcmd.Load(b)
	if err != nil {
		return xerrors.Errorf("loading config: %w", err)
	}

	for name, cluster := range cfg.Clusters {
		if err := setCluster(name, cluster); err != nil {
			shouldRollback = true
			return xerrors.Errorf("setting cluster info: %w", err)
		}
	}
	for name, authInfo := range cfg.AuthInfos {
		if err := setCredentials(name, authInfo); err != nil {
			shouldRollback = true
			return xerrors.Errorf("setting credentials: %w", err)
		}
	}
	for name, context := range cfg.Contexts {
		if err := setContext(name, context); err != nil {
			shouldRollback = true
			return xerrors.Errorf("setting context: %w", err)
		}
	}
	return nil
}

func getKubeconfigPath(given string) (string, error) {
	if given != "" {
		return given, nil
	}
	fromEnv := os.Getenv("KUBECONFIG")
	if fromEnv != "" {
		return fromEnv, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", xerrors.Errorf("getting ${HOME}: %w", err)
	}
	return filepath.Join(home, ".kube", "config"), nil
}

func backup(kubeconfig string) (io.Reader, error) {
	f, err := os.Open(kubeconfig)
	if err != nil {
		return nil, xerrors.Errorf("opening kubeconfig: %s: %w", kubeconfig, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return nil, xerrors.Errorf("reading kubeconfig: %s: %w", kubeconfig, err)
	}
	return &buf, nil
}

func rollback(r io.Reader, kubeconfig string) error {
	fmt.Println(">> Rollback")
	f, err := os.OpenFile(kubeconfig, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return xerrors.Errorf("opening kubeconfig: %s: %w", kubeconfig, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return xerrors.Errorf("writing kubeconfig: %s: %w", kubeconfig, err)
	}
	return nil
}

func setCluster(name string, cluster *api.Cluster) error {
	args := []string{"config", "set-cluster", name}
	if cluster.Server != "" {
		args = append(args, "--server="+cluster.Server)
	}
	if len(cluster.CertificateAuthority) != 0 {
		args = append(args, "--certificate-authority="+cluster.CertificateAuthority)
	}
	if cluster.InsecureSkipTLSVerify {
		args = append(args, "--insecure-skip-tls-verify=true")
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig="+opts.Kubeconfig)
	}
	return kubectl(args)
}

func setContext(name string, context *api.Context) error {
	args := []string{"config", "set-context", name}
	if context.Cluster != "" {
		args = append(args, "--cluster="+context.Cluster)
	}
	if context.AuthInfo != "" {
		args = append(args, "--user="+context.AuthInfo)
	}
	if context.Namespace != "" {
		args = append(args, "--namespace="+context.Namespace)
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig="+opts.Kubeconfig)
	}
	return kubectl(args)
}

func setCredentials(name string, authInfo *api.AuthInfo) error {
	args := []string{"config", "set-credentials", name}
	if authInfo.ClientCertificate != "" {
		args = append(args, "--client-certificate="+authInfo.ClientCertificate)
	}
	if authInfo.ClientKey != "" {
		args = append(args, "--client-key="+authInfo.ClientKey)
	}
	if authInfo.Token != "" {
		args = append(args, "--token="+authInfo.Token)
	}
	if authInfo.Username != "" {
		args = append(args, "--username="+authInfo.Username)
	}
	if authInfo.Password != "" {
		args = append(args, "--password="+authInfo.Password)
	}
	if authInfo.AuthProvider != nil {
		args = append(args, "--auth-provider="+authInfo.AuthProvider.Name)
		for key, value := range authInfo.AuthProvider.Config {
			args = append(args, "--auth-provider-arg="+key+"="+value)
		}
	}
	if opts.Kubeconfig != "" {
		args = append(args, "--kubeconfig="+opts.Kubeconfig)
	}
	return kubectl(args)
}

func kubectl(args []string) error {
	fmt.Printf("kubectl %s\n", strings.Join(args, " "))
	output, err := exec.Command("kubectl", args...).Output()
	if err != nil {
		return xerrors.Errorf("executing kubectl config set-cluster: %w", err)
	}
	fmt.Printf("%s\n", output)
	return nil
}
