WTF Dial ![GitHub release](https://img.shields.io/github/v/release/benbjohnson/wtf) ![test](https://github.com/benbjohnson/wtf/workflows/test/badge.svg) ![deploy](https://github.com/benbjohnson/wtf/workflows/deploy/badge.svg)
========

This project provides a real-time dashboard for teams to view how f-cked up they 
currently are. Each team member provides input to specify the level at which
they feel the team is currently messed up. These values range from 0% (meaning
team feels there are no WTF situations) to 100% (meaning the members feel the
team is completely f-cked).

The idea for this came from [Peter Bourgon's tweets][tweets].

[tweets]: https://twitter.com/peterbourgon/status/765935213507649537


## How to use this repository

This repository was built to help others learn how to build a fully functioning
Go application. It can be used in several ways:

1. As a reference—the code is well documented. Honestly, too documented for most
   projects but the goal here is to be as clear as possible for anyone reading
   the code.

2. As a walkthrough—companion blog posts will be added to the [Go Beyond](https://www.gobeyond.dev/)
   web site that walk through the various parts of the application and explain the design choices.
   You can find the initial blog post here: https://www.gobeyond.dev/wtf-dial/

You can also see the project structure overview below to get a quick overview
of the application structure.


## Project structure

The `wtf` project organizes code with the following approach:

1. Application domain types go in the root—`User`, `UserService`, `Dial`, etc.
2. Implementations of the application domain go in subpackages—`sqlite`, `http`, etc.
3. Everything is tied together in the `cmd` subpackages—`cmd/wtf` & `cmd/wtfd`.


### Application domain

The application domain is the collection of types which define what your
application does without defining how it does it. For example, if you were to
describe what WTF Dial does to a non-technical person, you would describe it in
terms of _Users_ and _Dials_.

We also include interfaces for managing our application domain data types which
are used as contracts for the underlying implementations. For example, we define
a `wtf.DialService` interface for CRUD (Create/Read/Update/Delete) actions and
SQLite does the actual implementation.

This allows all packages to share a common understanding of what each service
does. We can swap out implementations, or more importantly, we can layer
implementations on top of one another. We could, for example, add a Redis
caching layer on top of our database layer without having the two layers know
about one another as long as they both implement the same common interface.


### Implementation subpackages

Most subpackages are used as an adapter between our application domain  and the
technology that we're using to implement the domain. For example,
`sqlite.DialService` implements the `wtf.DialService` using SQLite.

The subpackages generally should not know about one another and should
communicate in terms of the application domain.

These are separated out into the following packages:

- `http`—Implements services over HTTP transport layer.
- `inmem`—Implements in-memory event listener service & subscriptions.
- `sqlite`—Implements services on SQLite storage layer.

There is also a `mock` package which implements simple mocks for each of the
application domain interfaces. This allows each subpackage's unit tests to share
a common set of mocks so layers can be tested in isolation.


### Binary packages

The implementation subpackages are loosely coupled so they need to be wired
together by another package to actually make working software. That's the job
of the `cmd` subpackages which produce the final binary.

There are two binaries:

- `wtfd`—the WTF server
- `wtf`—the client CLI application

Each of these binaries collect the services together in different ways depending
on the use case.

The `wtfd` server binary creates a `sqlite` storage layer and adds the `http`
transport layer on top. The `wtf` client binary doesn't have a storage layer.
It only needs the client side `http` transport layer.

The `cmd` packages are ultimately the interface between the application domain
and the operator. That means that configuration types & CLI flags should live
in these packages.


### Other packages

A few smaller packages don't fall into the organization listed above:

- `csv`—implements a `csv.DialEncoder` for encoding a list of Dial objects to
  a writer using the CSV format.
- `http/html`-groups together HTML templates used by the `http` package.



## Development

You can build `wtf` locally by cloning the repository and installing the 
[ego](https://github.com/benbjohnson/ego) templating library.

Then run:

```sh
$ make 
$ go install ./cmd/...
```

The `wtfd` server uses GitHub for authentication so you'll need to [create a 
new GitHub OAuth App](https://github.com/settings/applications/new).

Next, you'll need to setup a configuration file in `~/wtfd.conf`:

```toml
[github]
client-id     = "00000000000000000000"
client-secret = "0000000000000000000000000000000000000000"

[http]
addr      = ":3000"
block-key = "0000000000000000000000000000000000000000000000000000000000000000"
hash-key  = "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
```

Replace the GitHub `client-id` & `client-secret` with the values from the
GitHub OAuth application you registered.

The `[http]` section can be left as-is for a local environment. The key fields
need random hex values for generating secure cookies but all zeros is ok for
local testing.

Finally, run the `wtfd` server and open the web site at [`http://localhost:3000`](http://localhost:3000):

```
$ $GOPATH/bin/wtfd
```


### Storybook

The `wtf-storybook` binary allows you to test UI views with prepopulated data.
This can make it easier to quickly test certain scenarios without needing to 
set up your backend database.

To run storybook, simply build it and run it:

```sh
$ go install ./cmd/wtf-storybook
$ wtf-storybook
Listening on http://localhost:3001
```

To add a new view, add an entry to the `routes` variable:

```go
var routes = []*Route{
	// Show dial listing when user has no dials.
	{
		Name: "Dial listing with data",
		Path: "/dials-with-no-data",
		Renderer: &html.DialIndexTemplate{
			Dials: []*wtf.Dial{},
		},
	},
}
```

Then navigate to https://localhost:3001 and you'll see it displayed in the list.


### SQLite

By default, the SQLite tests run against in-memory databases. However, you can
specify the `-dump` flag for the tests to write data out to temporary files. This
works best when running against a single test.

```sh
$ go test -run=MyTest -dump ./sqlite
DUMP=/tmp/sy9j7nks0zq2vr4s_nswrx8h0000gn/T/375403844/db
```

You can then inspect that database using the `sqlite3` CLI to see its contents.


## Contributing

This application is built for educational purposes so additional functionality
will likely be rejected. Please feel free to submit an issue if you're
interested in seeing something added. Please do not simply submit a pull request.

