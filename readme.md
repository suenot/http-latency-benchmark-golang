# Bybit-ping-go

### Как получить из монги сервер с наименьшим пингом
```
db.getCollection("pings").aggregate([
    {
        $group: {
            _id: "$ip",  // Используем `ip` вместо `Ip`
            durations: { $push: "$durationms" }
        }
    },
    {
        $addFields: {
            sortedDurations: { $sortArray: { input: "$durations", sortBy: 1 } }
        }
    },
    {
        $addFields: {
            medianDuration: {
                $cond: {
                    if: { $eq: [{ $mod: [{ $size: "$sortedDurations" }, 2] }, 0] },
                    then: {
                        $avg: [
                            { $arrayElemAt: ["$sortedDurations", { $divide: [{ $size: "$sortedDurations" }, 2] }] },
                            { $arrayElemAt: ["$sortedDurations", { $subtract: [{ $divide: [{ $size: "$sortedDurations" }, 2] }, 1] }] }
                        ]
                    },
                    else: { $arrayElemAt: ["$sortedDurations", { $floor: { $divide: [{ $size: "$sortedDurations" }, 2] } }] }
                }
            }
        }
    },
    {
        $sort: { medianDuration: 1 }
    },
    {
        $limit: 1
    }
])
```

### пример ответа:
```
{
    "_id" : "111.111.111.111",
    "durations" : [
        808.888,
        283.368,
        270.865,
        620.161,
        267.611,
        608.134,
        255.33,
        550.301,
        627.726,
        266.174
    ],
    "sortedDurations" : [
        255.33,
        266.174,
        267.611,
        270.865,
        283.368,
        550.301,
        608.134,
        620.161,
        627.726,
        808.888
    ],
    "medianDuration" : 416.83450000000005
}
```