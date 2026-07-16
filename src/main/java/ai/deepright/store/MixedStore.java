package ai.deepright.store;

import ai.open.right.workflow.flow.file.impl.SysStore;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Slf4j
public class MixedStore extends SysStore {

    @ConditionalOnProperty(name = "file.store.sys.enable", havingValue = "true", matchIfMissing = true)
    @Configuration
    @Setter
    @Getter
    public static class MixedInitConfig extends InitConfig {

        @Bean(SysStore.NAME)
        // 存在S3则关闭FileSys
        @ConditionalOnMissingBean(name = SysStore.NAME)
        public MixedStore sysStore() throws Exception {
            MixedStore mixedStore = new MixedStore();
            BeanUtils.copyProperties(this, mixedStore);
            log.info("MixedStore inited: path={}", mixedStore.getPath());
            return mixedStore;
        }
    }
}
