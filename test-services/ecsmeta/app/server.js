const PORT = 80

const fs = require('fs')

const express = require('express')
const app = express()

app.get('/:filename', (req, res) => {
    const filename = req.params.filename + '.json'
    const contents = fs.readFileSync(filename)
    const parsed = JSON.parse(contents)

    if(filename == 'metadata.json') { // Lose a docker id to simulate one missing container in the metadata
        parsed['Containers'][0]['DockerId'] = 'masked!'
    }

    res.status(200).json(parsed)
})

app.get('/:filename/:key', (req, res) => {
    const filename = req.params.filename
    const keyname = req.params.key
    const contents = fs.readFileSync(filename + '.json')
    const parsed = JSON.parse(contents)

    switch(filename) {
        case 'metadata':
            const containers = parsed['Containers']

            for(let i = 0; i < containers.length; i++) {
                if(containers[i]['DockerId'] === keyname) {
                    res.status(200).json(containers[i])
                    break
                }
            }
            res.status(404).send('Key not found')
            break
        case 'stats':
            if(parsed[keyname]) {
                res.status(200).json(parsed[keyname])
            }
            else {
                res.status(404).send('Key not found')
            }
            break
    }
})

app.listen(PORT, () => {
    console.info(`ECS metadata endpoint is running on ${PORT}`)
})
