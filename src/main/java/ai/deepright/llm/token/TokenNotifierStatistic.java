package ai.deepright.llm.token;

import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.token.TokenData;
import ai.open.right.workflow.flow.llm.token.TokenStatistic;
import ai.open.right.workflow.flow.llm.token.impl.RedisTokenStatistic;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.collections.CollectionUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.List;

@Slf4j
@Setter
@Getter
public class TokenNotifierStatistic extends RedisTokenStatistic {

    protected List<TokenNotifier> notifiers;

    @Override
    public void stat(ProviderRequest request, TokenData tokenData) throws Exception {
        // 通知
        if (!CollectionUtils.isEmpty(this.notifiers)) {
            for (TokenNotifier notifier : this.notifiers) {
                try {
                    notifier.notify(request, tokenData);
                } catch (Exception e) {
                    WorkflowException.dolog(e);
                }
            }
        }
        super.stat(request, tokenData);
    }

    @Override
    protected String getModel(ProviderRequest request) throws Exception {
        return request.getModel();
    }

    @Configuration
    @Setter
    @Getter
    public static class CustomerInitConfig extends InitConfig {

        @Autowired(required = false)
        protected List<TokenNotifier> notifiers;

        @Override
        @Bean
        @ConditionalOnMissingBean(value = TokenStatistic.class)
        public TokenNotifierStatistic tokenStatistic() throws Exception {
            TokenNotifierStatistic tokenStatistic = new TokenNotifierStatistic();
            BeanUtils.copyProperties(this, tokenStatistic);
            log.info("TokenNotifierStatistic inited");
            return tokenStatistic;
        }
    }
}
