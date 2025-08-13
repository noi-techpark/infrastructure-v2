<!--
SPDX-FileCopyrightText: NOI Techpark <digital@noi.bz.it>

SPDX-License-Identifier: CC0-1.0
-->

# How to shovel data from one mongodb to another using mongoexport
Sometimes you need to shovel data around between dev and prod, for example if you go into production and you want to reuse the data in testing, or if you want to update testing with production data.

To do this, use the MongoDB compass gui, specify your query and then "Export Data".
This writes a json file, which in the other environment you can load via "Add data".

You have to make sure that the data doesn't overlap or duplicate yourself.