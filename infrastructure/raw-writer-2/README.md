<!-- 
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
 -->

# Raw data writer API for Open Data Hub
This service exposes a REST interface to POST raw data to the Open Data Hub data lake.

The single endpoint is:
```
POST /{provider1}/{provider2}/{timestamp}
```

Metadata is constructed via the url path parameters, and by parsing certain headers:
- `User-Agent` becomes the `provenance` field in raw data
- `X-OpenDataHub-xxxx` becomes `xxxx` field in raw data

## File size
The storage location of the raw data is based on the payload size.  
If the value is below threshold, messages are directly stored in mongodb in field `rawdata`

If the value is above, messages are pushed to an S3 compatible object storage. A reference with all the metadata is stored in mongodb as field `raw_ref`

## Content-Type
If the content type is some kind of binary format that is not represented well by text, the storage type will be as a binary.  
If the type is "texty" then it is interpreted as a string.

### Compression
The content type is also considered when deciding if a file should be compressed.  
Compressed files are stored in zstd format.  
Only files in object storage are compressed and the raw data bridge will decompress before delivering the raw data

### Bson
If the content type is set to `application/bson` and the file is below size threshold, the writer attempts to interpret it as a mongodb compabile object.  
I.e. the structure will be stored as a bson object inside the `rawdata` field


