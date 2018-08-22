# concourse-build-resource

Sometimes you want to get build information and logs out of Concourse so you can look at it elsewhere. 

This resource aims to make that as easy as possible.

## source

* `concourse_url`: the base URL for your Concourse.
* `team`, `pipeline`, `job`: hopefully self-explanatory

Authentication is not supported currently. 

Your pipeline and jobs will need to be public or it won't work.

## in

Will produce a number of files in the resource directory.

### The original responses

* `build.json`: the build metadata
* `resources.json`: the resource versions that were involved in `get`s or `put`s
* `plan.json`: the plan of the Job
* `events.log`: the logs from the Job.

### The original resources with information encoded in the filename

The same as the above, but with team, pipeline, job and job number added to the filename.
For example, as well as `build.json`, you would also get `build-teamname-pipelinename-jobname-123.json`.

This feature is intended to make it easier to `put` into blobstores using globs.

### Single-value files

Basic build data is extracted out of `build.json` and turned into individual files:

* `team`: the team name
* `pipeline`: the pipeline name
* `job`: the job name
* `global-number`: the build number from the sequence of all builds on a particular Concourse.
   This is the same value as the version itself, as it is unique across all teams, pipelines etc.
* `job-number`: the build number from the sequence for _this_ job. This appears in URLs and the
   web UI for single job builds. Not to be confused with `global-number`.
* `started-time`: Timestamp of when the build began.
* `ended-time`: Timestamp of when the build ended.
* `status`: The build status. Because this resource ignores `started` and `pending` builds, you will
   only see `succeeded`, `failed`, `errored` or `aborted`.
* `url`: the URL pointing to the original job's web UI. This is not the API URL you can find inside `build.json`.

### Warning

If you're scraping logs and other data, you may wind up collecting secrets and spreading them into a new location.
I can't prevent you from leaking secrets this way. Please make sure to keep sensitive pipelines and jobs private,
redact your logs and use proper credential management.

## out

No-op.

# Utility tasks

Some convenience tasks are included to help you make quick and easy use of the resource.

The default input to the tasks is `build`, but you can use 
[input mapping](https://concourse-ci.org/task-step.html#input_mapping) to rename this input.

## `build-pass-fail`

This task consumes the `build` folder output from the resource and itself passes or fails depending 
on the results of the build being watched.

This is useful if you coordinate with downstream teams who consume your work: you can add a job to your pipeline
which fails when the upstream fails.

## `show-build`, `show-plan`, `show-resources`

These tasks produce pretty-printed output of the build, plan and resource JSON files.

## `show-logs`

Produces the logs from the build, including colouring (which is retained in Concourse's logs).

To avoid confusion, the log being printed is wrapped with "begin log" and "end log" lines.

## Example

```yaml
resource_types:
- name: concourse-build
  type: docker-image
  source:
    repository: gcr.io/cf-elafros-dog/concourse-build-resource

resources:
- name: build
  type: concourse-build
  source:
    concourse_url: https://concourse.example.com
    team: main
    pipeline: example-pipeline
    job: some-job-you-are-interested-in

- name: concourse-build-resource
  type: git
  source: {uri: https://github.com/jchesterpivotal/concourse-build-resource.git}

jobs:
# ....

- name: some-job-you-are-interested-in
  public: true # required or the resource won't work
  plan:
  # ... whatever it is

- name: react-after-build
  public: true
  plan:
    - get: concourse-build-resource # for task YAML
    - get: build
      trigger: true
      version: every
    - task: pass-if-the-build-passed
      file: concourse-build-resource/tasks/build-pass-fail/task.yml
    - task: show-build
      file: concourse-build-resource/tasks/show-build/task.yml
    - task: show-plan
      file: concourse-build-resource/tasks/show-plan/task.yml
    - task: show-resources
      file: concourse-build-resource/tasks/show-resources/task.yml
    - task: show-logs
      file: concourse-build-resource/tasks/show-logs/task.yml
```

## Contributing

I'll be _very_ grateful if you write some tests first, given how much of a hassle it was to backfill them.

I'm using [spec](https://github.com/sclevine/spec) to organise the tests and
[Gomega](https://github.com/onsi/gomega) for matchers and utilities.
