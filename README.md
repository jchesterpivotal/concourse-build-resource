# concourse-build-resource

Right now this is untested, so if you use it in production, shame on you.

Basically it is a gaping, bleeding security hole. Untested and dangerous. 

Kick the tires, but if the tires explode and turn your feet into abstract art, I refuse to be held responsible.

## source

* `concourse_url`: the base URL for your Concourse.
* `team`, `pipeline`, `job`: hopefully self-explanatory

Authentication is not supported currently. 

Your pipeline and jobs will need to be public or it won't work.

## in

Will produce 4 files:

* `build.json`: the build metadata
* `resources.json`: the resource versions that were involved in `get`s or `put`s
* `plan.json`: the plan of the Job
* `events.log`: the logs from the Job.

**THERE IS NO REDACTION. THIS MEANS YOUR SECRETS MAY SHOW UP IN THESE FILES. BE CAREFUL.** 

## out

No-op.
