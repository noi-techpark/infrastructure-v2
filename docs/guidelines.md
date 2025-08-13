<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

## Guidelines

## Datacollectors provider URI standard

When setting a `provider` for a new datacollector, is important to follow the below rules to prevent confusion or inconsistent naming.

### 1. Provider key structure
The provider key is **ALWAYS** composed by at least 2 paths divided by `/`:

```
path1/path2
```

The provider key **CAN** be composed by additional paths divided by `/`:

```
path1/path2/path3
```

### 2. Provider key constraints
The `provider key` **MUST** be composed by characters which satisfy the below set:

```
[a-zA-Z1-9_\-\/]
```



### 3. Provider key semantic

The **first** path of the provider key **MUST** be the name of the collector `deployment`.  

The **second** path of the provider key **MUST** be the name of the collector `configuration`.  

If the same configuration has multiple `data-streams`, the `data-stream` name **SHOULD** be used as **third** path to identify the particular stream.

If the collector only has a single `configuration`, a name which identify the technical or contractual provider **MUST** be chosen and used.

If the collecot only has a single `configuration` but multiple `data-streams`, `data-stream` name **MUST** be used as **second** path.

## RawData URN Naming
To identify a raw data resource in the Open Data Hub we use [URN](https://datatracker.ietf.org/doc/html/rfc8141) standard.

Each resource is uniquely identify by an **URN** having the below format

```
urn:raw:provider:id
```

The provider follows the [above](#datacollectors-provider-uri-standard) standard replacing `/` with `:`

### Example:

Provider URI
```
provider_name/domain/subdomain
```

generates a reource
```
urn:raw:provider_name:domain:subdomain:id
```