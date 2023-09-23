# squat

Domain-Driven Design framework, event sourcing supported, base on EDA and CQRS.

## Feature List

- Event Sourcing (Developing)

  Save eventstream when aggregate changed. And also can been restored from eventstreams and snapshot.

  - Save `eventstream` to `eventstore`

  - Publish `eventstream` to `eventbus`

  - Support taking snapshot for `aggregate`

  - Record published eventstream to `publishedstore` when published to eventbus success

- EDA (Developing)

  Publish events and handle them. In event handler, you can also publish another events.

  - Support user-customized `proxy` for event handler

  - Support callback when event handled

  - Support parallel handling events by `event source id` or `event source type`

- CQRS (TODO)
