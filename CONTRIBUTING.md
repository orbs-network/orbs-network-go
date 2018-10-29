*This is work in progress, it will turn into a Contributor's Guide in time.*

## Design Principles

This section explains our way of dealing with common concerns in the project.

### Go Context
TBD
* *When to use: Graceful shutdown, as Key-Value store?*
* *Contexts hierarchy*

### Timers
TBD
* *We modified the default Go implementation*

### Synchronization
TBD
* *Channel vs mutex: guidelines for using each of them*

### Error handling
The Orbs platform is a self-healing eco-system. This means that returning Go `Error`s is only meaningful as a logging tool.
Human intervention should not be required to fix a condition that caused an `Error`.
#### Unrecoverable errors
In the event of an unrecoverable error, the app panics and crashes.
This includes assertions on conditions that cannot happen unless there is a software bug, system errors such as **Out of memory** etc.

### Logging
TBD
* *Our logging framework*


### Monitoring
TBD
* Implementation in code
* Tools that display data

### Configuration


### Performance testing
TBD

### Testing Strategy
Before
During
After

### CI
TBD

