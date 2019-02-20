"""
Integration tests for the ecs observer
"""
import time
from functools import partial as p
from textwrap import dedent
import string

from tests.helpers.assertions import has_datapoint_with_dim
from tests.helpers.util import ensure_always, run_agent, run_service, wait_for

CONFIG = string.Template(
    """
observers:
  - type: ecs
    metadataEndpoint: http://127.0.0.1/$case
    labelsToDimensions:
      mylabel: mydim

monitors:
  - type: collectd/redis
    discoveryRule: container_image =~ "redis"
  - type: collectd/mongodb
    discoveryRule: container_image =~ "mongo"
"""
)

def test_ecs_observer_single():
  with run_service("ecsmeta", name="ecsmeta-single", labels={"mylabel": "abc"}, ports={"80/tcp": ("127.0.0.1", 80)}) as ecsmeta :
    with run_service("redis", name="redis-single", labels={"mylabel": "abc"}, ports={"6379/tcp": ("127.0.0.1", 6379)}) as redis :
      with run_agent(CONFIG.substitute(
                case="metadata_single"
      )) as [backend, _, _]:
        assert wait_for(p(has_datapoint_with_dim, backend, "container_image", "redis:latest")), "Didn't get ecs metadata datapoints"

      ecsmeta.kill()
      ecsmeta.remove()
      redis.kill()
      redis.remove()

      # Let redis be removed by docker observer and collectd restart
      time.sleep(5)
      backend.datapoints.clear()
      assert ensure_always(lambda: not has_datapoint_with_dim(backend, "ClusterName", "seon-fargate-test"), 10)


# def test_ecs_observer_multi_containers():
#   with run_service("ecsmeta", name="ecsmeta-multi-containers", labels={"mylabel": "abc"}, ports={"80/tcp": ("127.0.0.1", 80)}) as ecsmeta :
#     with run_service("redis", name="redis-multi-containers", labels={"mylabel": "abc"}, ports={"6379/tcp": ("127.0.0.1", 6379)}) as redis, \
#     run_service("mongo", name="mongo-multi-containers", labels={"mylabel": "abc"}, ports={"27017/tcp": ("127.0.0.1", 27017)}) as mongo:
#       with run_agent(CONFIG.substitute(
#                 case="metadata_multi_containers"
#       )) as [backend, _, _]:
#         assert wait_for(p(has_datapoint_with_dim, backend, "container_image", "redis:latest")), "Didn't get ecs metadata datapoints"
#         assert wait_for(p(has_datapoint_with_dim, backend, "container_image", "mongo:latest")), "Didn't get ecs metadata datapoints"
      
#       ecsmeta.kill()
#       ecsmeta.remove()
#       redis.kill()
#       redis.remove()
#       mongo.kill()
#       mongo.remove()
      
#       # Let redis be removed by docker observer and collectd restart
#       time.sleep(5)
#       backend.datapoints.clear()
#       assert ensure_always(lambda: not has_datapoint_with_dim(backend, "ClusterName", "seon-fargate-test"), 10)

# def test_ecs_observer_multi_ports():
#   with run_service("ecsmeta", name="ecsmeta-multi-ports", labels={"mylabel": "abc"}, ports={"80/tcp": ("127.0.0.1", 80)}) as ecsmeta :
#     with run_service("redis", name="redis-multi-ports", labels={"mylabel": "abc"}, ports={"6379/tcp": ("127.0.0.1", 6379)}) as redis, \
#     run_service("mongo", name="mongo-multi-ports", labels={"mylabel": "abc"}, ports={"27017/tcp": ("127.0.0.1", 27017)}) as mongo:
#       with run_agent(CONFIG.substitute(
#                 case="metadata_multi_ports"
#       )) as [backend, _, _]:
#         assert wait_for(p(has_datapoint_with_dim, backend, "container_image", "redis:latest")), "Didn't get ecs metadata datapoints"
#         assert wait_for(p(has_datapoint_with_dim, backend, "container_image", "mongo:latest")), "Didn't get ecs metadata datapoints"
      
#       ecsmeta.kill()
#       ecsmeta.remove()
#       redis.kill()
#       redis.remove()
#       mongo.kill()
#       mongo.remove()
      
#       # Let redis be removed by docker observer and collectd restart
#       time.sleep(5)
#       backend.datapoints.clear()
#       assert ensure_always(lambda: not has_datapoint_with_dim(backend, "ClusterName", "seon-fargate-test"), 10)
