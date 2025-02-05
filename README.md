# 🥔 Mutato: Your Kubernetes Mutation Sidekick!

Welcome to `Mutato` – the dynamic, flexible, and slightly potato-themed tool you never knew your Kubernetes cluster needed! 🥔✨

Mutato is a lightweight yet powerful `MutatingWebhook` configuration tool for Kubernetes. Whether you're injecting sidecars, tweaking pod configurations, or morphing manifests on the fly, Mutato has you covered – all while keeping things simple and fun!

## 🚀 Features

* Dynamic Mutation: Add or modify resources in your pods effortlessly.
* Plug-and-Play: Set up and start mutating in minutes – no steep learning curves here!
* Flexible Rules: Customize webhook behaviors to match your specific needs.

## 🛠️ How It Works

1. Deploy Mutato into your cluster.
2. Define your mutation rules with REGO policy.
3. Let Mutato dynamically transform your pods like a pro.

## 🥔 Why the Name?

Like a potato, `Mutato` is:
* Simple but powerful.
* Always ready to adapt.
* A dependable addition to any Kubernetes recipe.


## 🌟 Get Started



### Install Mutato

```bash
helm upgrade --install -n extension-mutato --create-namespace mutato charts/mutato --debug --wait
```

### Create a Mutation Rule

```bash
kubectl apply -f examples/mutation-rule.yaml
```

### Deploy Deployment

```bash
kubectl create ns test
kubectl apply -f examples/deployment.yaml
```

### Verify Mutation

```bash
kubectl -n test get pods -o yaml
```