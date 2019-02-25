# Knative Build and Tekton Pipelines Steering Committee proposal

Jason Hall ([jasonhall@google.com](mailto:jasonhall@google.com))

Feb 25 2019

## Objective

Propose a resolution to the ongoing confusion and FUD around the future of Knative Build with regard to the recently announced Tekton Pipelines change. This doc should be used as a forum for external comment and questions, to resolve any concerns before the Steering Committee decides whether to adopt the proposal.

Please leave comments on this PR. The text of the proposal may change based on feedback, and the PR will retain the history of these edits.

## Background

Knative Build was conceived at the beginning of the Knative project alongside the other serverless-focused components (Serving and Eventing) to enable a "serverless development experience". Its goal was to enable users to specify their source to a Knative Service deployment and the steps to take to make it a container, and deploy that container, while interacting with a single custom resource.

From the beginning, Build had bigger ambitions. Its [2018 roadmap](https://github.com/knative/build/blob/master/roadmap-2018.md) mentions features that aren't directly related to furthering serverless deployments (triggering, workflow, etc.). These aren't accelerators for serverless deployments, these are CI/CD features!

This tension has existed in Build from the beginning: did it exist to accelerate the Knative serverless platform? Or is it supposed to be a full-featured CI/CD platform? How should Build prioritize features that don't have anything to do with the rest of Knative? These are not hypothetical questions, these are questions we've heard from users, operators and contributors.

To further add confusion, in August 2019 Knative began the Pipelines effort in the knative/build-pipeline repo. This was an experimental ground-up design to build on the experience with the Build resource, and build real CI/CD workflow infrastructure running on Kubernetes. Its [2019 roadmap](https://github.com/knative/build-pipeline/blob/master/roadmap-2019.md) (the knative/build repo never had a 2019 roadmap) included all the CI/CD features from 2018 and more, and makes only [passing mention](https://github.com/knative/build-pipeline/blob/master/roadmap-2019.md#dont-break-serving) of its relationship with Serving.

Around the same time, Knative Serving added duck-typing support, which meant any Kubernetes resource that satisfies its barebones interface could be used in a pre-deployment build, and all of Build, TaskRun and PipelineRun satisfy that interface and can be used.

Since the inception of the build-pipeline repo, most active development has happened there (with contributions from Pivotal, CloudBees, Red Hat and others). At this point, Knative Build is largely stable, requiring less than a single full-time engineer effort for requested features, high-priority bug fixes, and scheduled releases. This is not necessarily a bad thing! With the CI/CD features moving to Pipelines, there just aren't many serverless-focused features left for Build to implement at this time, and it could largely be considered "done" until it gets new feature requests from Knative's users and operators.

Pipelines contributors' ambitions continue to expand to address the market for CI/CD for deployment to platforms outside of Kubernetes: raw VMs, mobile devices, IoT devices, and the Kubernetes platform itself. Pipelines may primarily run _on Kubernetes_ but it's not necessarily _for Kubernetes_.

The vaguely overlapping goals and charters of the Build and Pipelines repos continues to cause confusion. Was one going away in favor of the other? Are we maintaining both indefinitely? Are we folding Pipelines into Build? Build into Pipelines? What's even going on. These are not hypothetical questions, these are questions we've heard from users, operators and contributors.

Googlers decided -- without much public input -- ahead of the Knative 0.4 release in February that Pipelines was a separate enough project to warrant its own versioning scheme and release cadence. We also decided (after lengthy trademark review) to christen it "Tekton Pipelines" instead of "Knative Pipelines" at this time.

This was an attempt to signal the Pipelines project's separateness, and help resolve the confusion around Pipelines' relationship with the rest of Knative. Confusion has remained. :)

The above decisions have regrettably all been made and would be difficult (and I would argue even more confusing) to walk back. We can only hope to make better more publicly-informed decisions going forward. This proposal is an attempt to do that. Everything below this paragraph is a **proposal** and is open for discussion. Please leave comments and ask questions.


## Proposal


### Knative Build

https://github.com/knative/build will be supported as it is for the foreseeable future, overseen by the Knative Build WG which will hold weekly meetings for the foreseeable future. It will continue to do releases co-versioned and timed with the other Knative components.

 \
Its charter and roadmap (which we will write with community input) will focus on enabling better serverless deployment experiences -- source-to-deployment scenarios -- and no more. No workflow, no triggering, just a better developer experience for the Knative serverless platform.

When someone asks, "is Knative's serverless developer experience good?" we should be able to say, "yes, because Knative Build focuses on exactly that."

If you want automatic triggered CI/CD or build-test-rollout scenarios, I have good news for you...


### Tekton Pipelines

https://github.com/knative/build-pipeline should continue its move to a new GitHub org, to https://github.com/tektoncd/pipeline.

Its charter and roadmap should continue to focus on building infrastructure to run CI/CD workloads, including triggering, workflow, resources, retries, the works. It should not focus on enabling better serverless deployments specifically, but better continuous deployments to any target generally.

When someone asks, "is there a way to run CI/CD on Kubernetes?" we should be able to say, "yes, because Tekton Pipelines focuses on exactly that."

Until separate Tekton governance is established (weeks, not months), Tekton Pipelines work will continue to be overseen by the Knative Build WG -- largely by Christie (@, as she has been doing a great job leading Pipelines work so far -- though this will likely cause confusion and should be resolved as soon as possible. How we resolve that is TBD, and I think that's outside the scope of this proposal.


## Future Directions

There may come a time in the future when we decide that maintaining Knative Build and Tekton Pipelines as separate efforts is not the best use of engineering resources, and we have in the past discussed the possibility of reimplementing Build as a wrapper around Tekton TaskRuns, or having Tekton Pipelines produce a library that Knative Build could use to avoid duplicated code and maintenance. _Neither of these is currently a plan of record_, and before we pursue these or any other future structural changes to Build, those will be proposed and adopted separately in the future. These are only mentioned here as possible future directions worth noting.


## Alternatives Considered


### Keep Pipelines in Knative

The Knative brand is strong, and staying in Knative has the benefit of inertia; all we have to do is..._nothing_.

Ultimately, however, I don't think this resolves enough confusion. Knative Build and Knative Pipelines _sound_ very similar, but they serve two very different purposes. There's a resource in the Pipelines repo that isn't Build but looks and acts very much like one, and will continue to grow features wholly unrelated to serverless deployments -- which should we direct users to? This is the problem we have today, and doing nothing does nothing to solve it.

Knative's brand is also heavily associated with "serverless" -- serverless on-demand request serving, autoscaling to zero, event consumption and developer experience -- and while this is great for Knative's branding and positioning in the market, if we expand Knative to include CI/CD primitives then we'll have to reframe what Knative means in people's minds, which won't be easy.

In the end, I think having a separate term for "CI/CD _on_ K8s" and "Serverless _for_ K8s" will be valuable to both projects.
