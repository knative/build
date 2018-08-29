## Examples

This directory contains examples of [the Pipeline strawman CRDs](../README.md) in action.

### Example Tasks

* Example [Tasks](../README.md#task) are in:
  * [build_task.yaml](build_task.yaml) 
  * [deploy_tasks.yaml](deploy_tasks.yaml)
  * [test_tasks.yaml](test_tasks.yaml)

Here are the Task Types that are defined.

1. `build-push`: This task as the name suggests build an image via kaniko and pushes it to registry.
2. `make`:  Runs make target.
3. `integration-test-in-docker`: This is a new task that is used in the sample pipelines to test an app in using `docker build` command to build an image with has the integration test code.
This task then calls `docker run` which will run the test code. This follows the steps we have for [kritis integration test](https://github.com/grafeas/kritis/blob/4f83f99ca58751c28c0ec40016ed0bba5867d70f/Makefile#L152)
4. `deploy-with-helm`: This task deploys a kubernetes app with helm.
5. `deploy-with-kubectl`: This task deploys with kubectl apply -f <filename>

#### Example Task Run

The [runs](./runs/) dir contains an example [TaskRuns](../README.md#taskrun) that runs `TaskRun`.

The [run-kritis-test.yaml](./invocations/run-kritis-test.yaml) shows an example how to manually run kritis unit test off your development branch.

### Example Pipelines

Finally, we have 2 example [Pipelines](../README.md#pipeline) in [./pipelines](./pipelines)

1. [Kritis](.pipelines/kritis.yaml): This exmaple demonstrates how to configure a pipeline which runs  unit test, build an image, deploys it to test and then run integration tests.

![Pipeline Configuration](./pipelines/kritis-pipeline.png)

2. [Guestbook](./pipelines/guestbook.yaml): This is pipeline which is based on example application in [Kubernetes example Repo](https://github.com/kubernetes/examples/tree/master/guestbook)
This pipeline demonstartes how to integrate frontend [guestbook app code](https://github.com/kubernetes/examples/tree/master/guestbook-go) with backed [redis-docker image](https://github.com/GoogleCloudPlatform/redis-docker/tree/master/4) provided by GCP.

![Pipeline Configuration](./pipelines/guestbook-pipeline.png)
