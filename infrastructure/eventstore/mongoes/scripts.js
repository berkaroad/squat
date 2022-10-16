// eventstreams
db.createCollection("eventstreams")
db.getCollection("eventstreams").createIndex({ "aggregateid": 1, "streamversion": 1 }, { "name": "idx_aggregateid_streamversion", "unique": true })

// aggregatesnapshots
db.createCollection("aggregatesnapshots")
db.getCollection("aggregatesnapshots").createIndex({ "aggregateid": 1 }, { "name": "idx_aggregateid", "unique": true })
