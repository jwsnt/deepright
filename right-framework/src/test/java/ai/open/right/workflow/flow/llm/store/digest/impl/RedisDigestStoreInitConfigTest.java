package ai.open.right.workflow.flow.llm.store.digest.impl;

import org.junit.Assert;
import org.junit.Test;

public class RedisDigestStoreInitConfigTest {

    @Test
    public void shouldCreateRedisDigestStore() throws Exception {
        RedisDigestStore.InitConfig init = new RedisDigestStore.InitConfig();

        RedisDigestStore bean = (RedisDigestStore) init.digestStore();

        Assert.assertNotNull(bean);
        Assert.assertTrue(bean instanceof RedisDigestStore);
    }
}
