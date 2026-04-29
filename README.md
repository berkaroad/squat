# squat

Domain-Driven Design framework, event sourcing supported, base on EDA and CQRS.

## I. Architecture Design

* 1. Data Flow Diagram

  ```mermaid
  sequenceDiagram
      participant C as Client
      participant CS as CommandService
      participant CD as CommandDispatcher
      participant CH as CommandHandlers
      participant R as Repository
      participant ES as EventStore
      participant EB as EventBus
      participant ED as EventDispatcher
      participant EH as EventHandlers
      participant RM as ReadModel
  
      C->>CS: send command
      activate CS
      CS->>CD: transport command
      activate CD
      CD->>CH: dispatch command
      activate CH
  
      alt Business Logic Checking Success
          CH->>CH: business logic checking, execute business logic
          CH->>R: save aggregate state
          activate R
          R->>ES: save eventstream
          activate ES
          ES-->>R: confirm of save success 
          deactivate ES
          R->>EB: push to event bus
          activate EB
          R-->>CH: confirm of save success 
          deactivate R
          CH-->>CD: handle command complete
          deactivate CH
          CD->>CS: notify that handle command success
          deactivate CD
          CS-->>C: execute command success from command
          deactivate CS
          EB->>ED: transport event
          activate ED
          deactivate EB
          ED->>EH: dispatch event
          activate EH
          EH->>EH: handle event
          EH->>RM: update read-model
          activate RM
          RM-->>EH: confirm of update success
          deactivate RM
          EH-->>ED: handle event complete
          deactivate EH
          deactivate ED
      else Business Logic Checking Fail
          activate CH
          CH-->>CD: return error
          activate CD
          deactivate CH
          CD->>CS: notify that handle command fail
          activate CS
          deactivate CD
          CS-->>C: execute command fail
          deactivate CS
      end
      
      C->>RM: query data
      activate RM
      RM-->>C: return data
      deactivate RM
  
      Note right of C: process of CQRS and eventsourcing is end
  ```

## II. Feature List

* 1. Event Sourcing

  Save eventstream when aggregate changed. And also can been restored from eventstreams and snapshot.

  * 1) Save `eventstream` to `eventstore`
  
  * 2) Publish `eventstream` to `eventbus`

  * 3) Take snapshot for `aggregate` to `snapshotstore`

* 2. EDA

  Publish events and handle them. In event handler, you can also publish another events.

  * 1) Support user-customized `proxy` for event handler

  * 2) Support notify when command-id related eventstream handled

  * 3) Support parallel handling events by different mailbox's name

  * 4) Record published eventstream to `publishedstore` when published to eventbus success

  * 5) Support parallel handling same event with different event handlers

* 3. CQRS

  Send command to command service and returns two results: one is when command handled, the other is when command-id related eventstream handled.

  * 1) Support user-customized `proxy` for command handler

  * 2) Support notify when command handled

  * 3) Support parallel handling commands by different mailbox's name

  * 4) Support process manager, for communication with multiple aggregate instances
