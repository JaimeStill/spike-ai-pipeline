// Package apptest holds the fixtures internal/docproc/app's tests share for running docproc over
// the composition [app.App] publishes: its graph and its nodes. A test builds its App with
// [app.New] and applies the fixtures to it before it runs, each an observer or a Replace on the
// App's graph; a constructor a fixture does not swap stays the production one.
//
// [Builds] records the name of every node a run's Build begins. [Halt] fails the lifecycle
// configuration node, so a run of a command that declares nodes fails before anything starts.
// [Tools] replaces the tool lookup, so a run finds only the programs a test gives it, and
// [NoTools] is a lookup that finds none.
//
// The package exports:
//
//   - [Recorder], the events a test records, in order, and its [Recorder.Record] and
//     [Recorder.Log]
//   - [Builds], which returns a Recorder of the nodes the App's runs build
//   - [Halt], which stops the App's runs before anything starts, and [ErrHalted], what such a
//     run fails with
//   - [Tools], which makes the App's runs find tools through a lookup the test gives, and
//     [NoTools], the lookup of a host with no tool at all
package apptest
