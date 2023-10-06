# squat

Domain-Driven Design framework, event sourcing supported, base on EDA and CQRS.

## Feature List

- Event Sourcing

  Save eventstream when aggregate changed. And also can been restored from eventstreams and snapshot.

  - Save `eventstream` to `eventstore`

  - Publish `eventstream` to `eventpublisher`

  - Support taking snapshot for `aggregate` to `snapshotstore`

- EDA

  Publish events and handle them. In event handler, you can also publish another events.

  - Support user-customized `proxy` for event handler

  - Support notify when command-id related eventstream handled

  - Support parallel handling events by specific mailbox's name

  - Record published eventstream to `publishedstore` when published to eventbus success

- CQRS

  Send command to command bus and returns two results: one is when command handled, the other is when command-id related eventstream handled.

  - Support user-customized `proxy` for command handler

  - Support notify when command handled

  - Support parallel handling commands by specific mailbox's name
