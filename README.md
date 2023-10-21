# squat

Domain-Driven Design framework, event sourcing supported, base on EDA and CQRS.

## Feature List

* 1. Event Sourcing

  Save eventstream when aggregate changed. And also can been restored from eventstreams and snapshot.

  * 1) Save `eventstream` to `eventstore`

  * 2) Publish `eventstream` to `eventpublisher`

  * 3) Support taking snapshot for `aggregate` to `snapshotstore`

* 2. EDA

  Publish events and handle them. In event handler, you can also publish another events.

  * 1) Support user-customized `proxy` for event handler

  * 2) Support notify when command-id related eventstream handled

  * 3) Support parallel handling events by different mailbox's name

  * 4) Record published eventstream to `publishedstore` when published to eventbus success

* 3. CQRS

  Send command to command bus and returns two results: one is when command handled, the other is when command-id related eventstream handled.

  * 1) Support user-customized `proxy` for command handler

  * 2) Support notify when command handled

  * 3) Support parallel handling commands by different mailbox's name

  * 4) Support process manager, for communication with multiple aggregate instances
