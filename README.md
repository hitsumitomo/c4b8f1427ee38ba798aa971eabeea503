# Storage Manager

This project implements a distributed storage system with a manager server and multiple storage servers. The manager server coordinates the operations of the storage servers, overseeing file distribution and management. Each storage server is responsible for handling file uploads, downloads, and local storage management. Together, they enable efficient, scalable file storage and retrieval across the distributed network.

## Updates

- The code has been **refactored** according to the [Go Project Layout](https://github.com/golang-standards/project-layout).
![Source tree](https://drive.google.com/uc?export=view&id=11wLiZBfDBe-OWSmHURW-4XNH598jSZDB)

## Features

- Upload and download files: Users can efficiently upload files to the system and retrieve them when needed.
- Distribute files across storage servers: Files are automatically distributed among multiple storage servers to ensure load balancing and redundancy.
- Retrieve files from distributed storage servers: The system supports downloading files from the appropriate storage servers, even in a distributed environment.
- Add new storage servers dynamically: New storage servers can be registered and integrated into the system seamlessly to scale storage capacity.
- Monitor storage usage: The system provides tools to track and manage storage utilization across all connected servers.

## Setup

1. **Clone the repository**

    ```bash
    git clone https://github.com/hitsumitomo/c4b8f1427ee38ba798aa971eabeea503.git
    ```
2. **Navigate to the project directory**

    ```bash
    cd c4b8f1427ee38ba798aa971eabeea503
    ```
3. **Update the environment variables in the `docker-compose.yml` file as needed, ensure your server has AVX support, then start the application using Docker Compose:**

    ```bash
    docker compose up
    ```
## Usage examples
**upload file**
```bash
curl -T data.bin http://localhost:18080
```
**download file**
```bash
curl http://localhost:18080/data.bin -o data2.bin
```
## MongoDB data storage
```bash
mongosh --port 19999 storage
storage> db.files.find()
```
```json
[
    {
        "_id": "673f9866d3649917678feffd",
        "name": "1diff.txt",
        "hash": "4376331571702d0bbab45fbc3a800180a478f146d44cf1dd545061ba5502ef31"
    },
    {
        "_id": "673f992b94d65446e7591f87",
        "name": "1.txt",
        "hash": "6c9db75e64e7237f4779d08d33eba68d71c5db2390aaf87b38310b42852b87fe"
    },
    {
        "_id": "673f992f94d65446e7591f89",
        "name": "2.txt",
        "hash": "6c9db75e64e7237f4779d08d33eba68d71c5db2390aaf87b38310b42852b87fe"
    },
    {
        "_id": "673f993094d65446e7591f8a",
        "name": "3.txt",
        "hash": "6c9db75e64e7237f4779d08d33eba68d71c5db2390aaf87b38310b42852b87fe"
    },
    {
        "_id": "673f9a2d8b722dda109cd04f",
        "name": "1.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9ad4d83efa4eed93b297",
        "name": "2.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9ad6d83efa4eed93b298",
        "name": "3.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9b73cc9d91ba49812d4f",
        "name": "4.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9d08cc9d91ba49812d50",
        "name": "5.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9e2f9f6b31f0d462d971",
        "name": "6.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9fcc9f6b31f0d462d972",
        "name": "7.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673f9ff69f6b31f0d462d973",
        "name": "8.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    },
    {
        "_id": "673fa0349f6b31f0d462d974",
        "name": "9.tgz",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180"
    }
]
```
```bash
storage> db.metadata.find()
```
```json
[
    {
        "_id": "673f9866d3649917678feffe",
        "hash": "4376331571702d0bbab45fbc3a800180a478f146d44cf1dd545061ba5502ef31",
        "size": 1048579,
        "metadata": [
            "http://172.18.0.7:19010/[STORED]/bf2b76f2c5f31b141b69403a58202ff654913b91dfbb22daf639856fed4d79fd",
            "http://172.18.0.10:19004/[STORED]/8841455851041340440508baff680d4b16ac7a4e6c86a7ddf0b6a88bcb092084",
            "http://172.18.0.4:19002/[STORED]/771b8f8c66622360e224faca1678a42d91744693638336a25ad1b389ad7ebfbf",
            "http://172.18.0.5:19001/[STORED]/6d8cf0318ff891ffd533a90bfd5c327356919c3b9e082ce20292c08da16ba6c1",
            "http://172.18.0.6:19003/[STORED]/f07cc53fcad5aff6d03451e6dd4ed3486aea0721a7dd8f71a3651f65a47371d0",
            "http://172.18.0.9:19005/[STORED]/32a4a218b3ebcdb3b43dba52fe25db3a7f37edb86428477ee53418bb05c5feae",
            "http://172.18.0.8:19000/[STORED]/7623ee8c5860caa8bd0163b159a31606a93eb364fe7be90dcd6768c28cb5db09"
        ]
    },
    {
        "_id": "673f992b94d65446e7591f88",
        "hash": "6c9db75e64e7237f4779d08d33eba68d71c5db2390aaf87b38310b42852b87fe",
        "size": 1048576,
        "metadata": [
            "http://172.18.0.9:19010/[STORED]/bf2b76f2c5f31b141b69403a58202ff654913b91dfbb22daf639856fed4d79fd",
            "http://172.18.0.4:19000/[STORED]/8841455851041340440508baff680d4b16ac7a4e6c86a7ddf0b6a88bcb092084",
            "http://172.18.0.8:19004/[STORED]/771b8f8c66622360e224faca1678a42d91744693638336a25ad1b389ad7ebfbf",
            "http://172.18.0.5:19002/[STORED]/6d8cf0318ff891ffd533a90bfd5c327356919c3b9e082ce20292c08da16ba6c1",
            "http://172.18.0.10:19005/[STORED]/f07cc53fcad5aff6d03451e6dd4ed3486aea0721a7dd8f71a3651f65a47371d0",
            "http://172.18.0.7:19001/[STORED]/32a4a218b3ebcdb3b43dba52fe25db3a7f37edb86428477ee53418bb05c5feae",
            "http://172.18.0.6:19003/[STORED]/23ad6fcd2118adb70af2ca65e125a2c82f639ac91bf3cd9d3460718155d2ac9f"
        ]
    },
    {
        "_id": "673f9a2d8b722dda109cd050",
        "hash": "920bcf444489afd8dc69977399caf1bce07125dc95e0ba715016fe5bc9df1180",
        "size": 16035511,
        "metadata": [
            "http://172.18.0.9:19010/[STORED]/94d4ead18fefe0047edee1aaf4221fbbc8a5a8236104787cefa2eab6477b2e93",
            "http://172.18.0.10:19000/[STORED]/dfe4a50240d2dccc65bf1cf1fe76b13e6d31d596db9acdf739f88040275af321",
            "http://172.18.0.4:19002/[STORED]/66ae3384fa0bf62dd8fe644ba89703cd50931a7c5e1f5e4b409bf0f3d8482816",
            "http://172.18.0.7:19003/[STORED]/5c1302b4727f1fae744d4a9c6afd71fdfab35b95b540b93ef1281a0e1c999555",
            "http://172.18.0.6:19004/[STORED]/03599d98f88abdc992924b94bcae5bf0f37a1ef09d06c7fccedcfa1ad51cc44c",
            "http://172.18.0.5:19005/[STORED]/9b178750ee91e0044b3a352eedfcc3206a659c023bd700b5a4c588c2ababea9d",
            "http://172.18.0.8:19001/[STORED]/0af9466c9bf4e561e86a0ee4e53efbd1d4c4d4794965a26250da9115e32f3f11"
        ]
    }
]
```

## Monitoring storages
```bash
curl http://localhost:18080/usage
```
```json
[
  {
    "Limit": 10737418240,
    "Used": 2760707,
    "URL": "http://172.18.0.5:19000"
  },
  {
    "Limit": 10737418240,
    "Used": 2912256,
    "URL": "http://172.18.0.9:19003"
  },
  {
    "Limit": 10737418240,
    "Used": 2903735,
    "URL": "http://172.18.0.7:19001"
  },
  {
    "Limit": 10737418240,
    "Used": 139267,
    "URL": "http://172.18.0.6:19010"
  },
  {
    "Limit": 10737418240,
    "Used": 2916352,
    "URL": "http://172.18.0.8:19004"
  },
  {
    "Limit": 10737418240,
    "Used": 2924544,
    "URL": "http://172.18.0.10:19005"
  },
  {
    "Limit": 10737418240,
    "Used": 2772992,
    "URL": "http://172.18.0.4:19002"
  }
]
```

