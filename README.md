# concourse-build-resource

Sometimes you want to get build information and logs out of Concourse so you can look at it elsewhere. 

This resource aims to make that as easy as possible.

Authentication is _not_ supported currently. You will need to make all the jobs and pipelines you are following public, or it (essentially) won't work.

### About this document

If you are reading the document on master, you will often see features described that are not in a release version.
Typically this means that they won't work in your pipeline if you are using `latest`, as I control release tags
fairly tightly.

[The latest release is `v0.8.0`](https://github.com/jchesterpivotal/concourse-build-resource/releases/tag/v0.8.0). See [the release README](https://github.com/jchesterpivotal/concourse-build-resource/tree/v0.8.0/README.md)
for features you can expect to see in production.

## source

* `concourse_url`: the base URL for your Concourse (Required)
* `team`: the team to follow. (Optional)
* `pipeline`: the pipeline to follow. (Optional)
* `job`: the job to follow (Optional)
* `initial_build_id`: the first build ID to start versions from, if you wish to start from an earlier build than
  the most recent on the target Concourse. Please note that if you set this to a very early version, you may wind
  up adding a  lot of builds for your local Concourse to churn through. (Optional)
* `fetch_page_size`: the maximum number of builds that can be fetched in a single `check`. (Optional, default 100)

If you leave off `job`, `pipeline` and/or `team`, concourse-build-resource will try to perform checks against whole
pipelines, or whole teams, or whole Concourse installations, respectively.

For example, this configuration watches all jobs in all pipelines in the `example` team:

```yaml
source:
  concourse_url: https://example.com/
  team: example-team
```

## in

Will produce a number of files in the resource directory.

### The original responses

* `build.json`: the build metadata
* `resources.json`: the resource versions that were involved in `get`s or `put`s
* `plan.json`: the plan of the Job
* `job.json`: the Job structure
* `events.log`: the logs from the Job.

### The original resources with information encoded in the filename

There are two variations.

* Detailed: Team, pipeline, job and job number are embedded in the filename.
  For example: `build_teamname_pipelinename_jobname_123.json`. Files use snake_case to create distinction with
  the kebab-case commonly used for pipeline and job names.
* Global build number: The global build number (unique across the Concourse instance) is embedded in the filename.
  For example: `build_9876.json`

This feature is intended to make it easier to `put` into blobstores using globs or regexps.

### Single-value files

Basic build data is extracted out of `build.json` and turned into individual files:

* `team`: the team name
* `pipeline`: the pipeline name
* `job`: the job name
* `global_number`: the build number from the sequence of all builds on a particular Concourse.
   This is the same value as the version itself, as it is unique across all teams, pipelines etc.
* `job_number`: the build number from the sequence for _this_ job. This appears in URLs and the
   web UI for single job builds. Not to be confused with `global_number`.
* `started_time`: Timestamp of when the build began.
* `ended_time`: Timestamp of when the build ended.
* `status`: The build status. Because this resource ignores `started` and `pending` builds, you will
   only see `succeeded`, `failed`, `errored` or `aborted`.
* `concourse_url`: the URL pointing to the original job's Concourse server. This will be the same as the `concourse_url`
  you set in `source`.
* `team_url`: the URL pointing to the team the pipeline belongs to.
* `pipeline_url`: the URL pointing to the pipeline the job belongs to.
* `job_url`: the URL pointing to the job the build belongs to.
* `build_url`: the full build URL for this build.

### in metadata

The resource injects metadata about itself into each JSON file under the `concourse_build_resource` key:

* `release`: The release version of the resource.
* `git_ref`: A shortref for the git commit the resource was built from.
* `get_timestamp`: A timestamp for when the actual `get` step (ie, the execution of `in`) took place. The timestamp
  is generated during the launch of `in` -- it reflects the start time of fetching data, not the end time. It also
  makes no attempt to be clever about timezones, so keep an eye out for those unwanted epoch dates.
* `concourse_version`: The version of Concourse the resource interacted with at the time of the `get`.

For consistency, these individual files contain the same information as the metadata injected into JSON:

* `concourse_build_resource_release`: Same information as `release` in the JSON files.
* `concourse_build_resource_git_ref`: Same information as `git_ref` in the JSON files.
* `concourse_build_resource_get_timestamp`: Same information as the `get_timestamp` in the JSON files.
* `concourse_version`: Same information as the `concourse_version` in the JSON files.

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
which fails when the downstream job fails.

## `show-build`, `show-plan`, `show-resources`, `show-job`

These tasks produce pretty-printed output of the build, plan, resource and job JSON files.

## `show-logs`

Produces the logs from the build, including colouring (which is retained in Concourse's logs).

To avoid confusion, the log being printed is wrapped with "begin log" and "end log" lines.

## Example

```yaml
resource_types:
- name: concourse-build
  type: docker-image
  check_every: 30m
  source:
    repository: jchesterpivotal/concourse-build-resource
    tag: v0.8.0 # check https://github.com/jchesterpivotal/concourse-build-resource/releases

resources:
- name: build
  type: concourse-build
  check_every: 30m # try to be neighbourly
  source:
    concourse_url: https://concourse.example.com
    team: main
    pipeline: example-pipeline
    job: some-job-you-are-interested-in
    initial_build_id: 12345

- name: concourse-build-resource # to retrieve utility task YAML
  type: git
  source:
    uri: https://github.com/jchesterpivotal/concourse-build-resource.git
    tag: v0.8.0 # check https://github.com/jchesterpivotal/concourse-build-resource/releases

jobs:
# ....

- name: some-job-you-are-interested-in
  public: true # when the target job is not public, this resource can't get its data
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
    - task: show-resources
      file: concourse-build-resource/tasks/show-job/task.yml
    - task: show-logs
      file: concourse-build-resource/tasks/show-logs/task.yml
```

## Versioning

* Each tagged release on Github corresponds to an image with the same tag.
* The container tagged `latest` points to the latest release.
* There are a bunch of `-rc` images that get built upon each commit. It's possible to guess at these tags, but I wouldn't recommend it.

It is safe to use `latest`, though I recommend pinning to versions so that your pipeline's history is clearer.

## Contributing

I'll be _very_ grateful if you write some tests first, given how much of a hassle it was to backfill them.

I'm using [spec](https://github.com/sclevine/spec) to organise the tests and
[Gomega](https://github.com/onsi/gomega) for matchers and utilities.
