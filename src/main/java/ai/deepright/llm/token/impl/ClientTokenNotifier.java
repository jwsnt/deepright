package ai.deepright.llm.token.impl;

import ai.deepright.cli.CliPubData;
import ai.deepright.cli.CliSubFetcher;
import ai.deepright.cli.CliSubOps;
import ai.deepright.feature.FeatureFlag;
import ai.deepright.feature.FeatureUtils;
import ai.deepright.lang.XmlResourceLang;
import ai.deepright.llm.token.TokenNotifier;
import ai.deepright.llm.token.TokenSource;
import ai.open.right.WorkflowException;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.token.TokenData;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.util.Assert;

import java.util.concurrent.ExecutorService;

@Getter
@Setter
@Slf4j
public class ClientTokenNotifier implements TokenNotifier, TokenSource {

    public static final String LANG_KEY_TOKEN_KNOWLEDGE = "token.knowledge";

    public static final String LANG_KEY_TOKEN_PROFILE = "token.profile";

    public static final String LANG_KEY_TOKEN_SKILL = "token.skill";

    public static final String LANG_KEY_TOKEN_CRON = "token.cron";

    public static final String LANG_KEY_TOKEN_TASK = "token.task";

    public static final String LANG_KEY_TOKEN_CHAT = "token.chat";

    protected static final String NAME = "token_notifier_client";

    protected ExecutorService executorService;

    protected CliSubFetcher cliSubFetcher;

    @Override
    public void notify(ProviderRequest request, TokenData tokenData) throws Exception {
        this.executorService.execute(NotifierRunnable.builder()
                .cliSubFetcher(this.cliSubFetcher)
                .providerRequest(request)
                .tokenData(tokenData)
                .tokenSource(this)
                .build());
    }

    @Override
    public String source(WorkflowTask workTask) throws Exception {
        if (FeatureFlag.isKnowledgeCommit(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_KNOWLEDGE);
        } else if (FeatureFlag.isProfileCommit(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_PROFILE);
        } else if (FeatureFlag.isSkillExtract(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_SKILL);
        } else if (FeatureFlag.isTask(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_TASK);
        } else if (FeatureFlag.isCron(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_CRON);
        } else {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_CHAT);
        }
    }

    @Builder
    public static class NotifierRunnable implements Runnable {

        protected ProviderRequest providerRequest;

        protected CliSubFetcher cliSubFetcher;

        protected TokenSource tokenSource;

        protected TokenData tokenData;

        @Override
        public void run() {
            try {
                String agentId = FeatureUtils.buildAgentId(this.providerRequest.getMessage());
                String function = this.tokenSource.source(this.providerRequest.getMessage());
                String app = FeatureUtils.buildApp(this.providerRequest.getMessage());
                String model = this.providerRequest.getModel();
                Integer thinking = this.tokenData.getThinking();
                Integer cache = this.tokenData.getCache();
                Integer input = this.tokenData.getInput();
                Integer total = this.tokenData.getTotal();
                Assert.hasText(function, "The function can not be empty");
                Assert.notNull(thinking, "The thinking can not be empty");
                Assert.hasText(agentId, "The agent id can not be empty");
                Assert.notNull(input, "The input can not be empty");
                Assert.notNull(total, "The total can not be empty");
                Assert.hasText(model, "The model can not be empty");
                Assert.hasText(app, "The app can not be empty");
                StringBuffer buffer = new StringBuffer().append(app).append(" token ");
                buffer.append(" --function ").append(function);
                buffer.append(" --thinking ").append(thinking);
                buffer.append(" --agentId ").append(agentId);
                buffer.append(" --cache ").append(cache);
                buffer.append(" --input ").append(input);
                buffer.append(" --total ").append(total);
                buffer.append(" --model ").append(model);
                if (log.isInfoEnabled()) {
                    log.info("The token cmd={}", buffer.toString());
                }
                CliPubData pubData = this.cliSubFetcher.command(this.providerRequest.getMessage(), CliSubOps.builder()
                        .exempted(true)
                        .build(), buffer.toString(), "");
                Assert.isTrue(pubData.isOk(), pubData.getCmd());
            } catch (Exception e) {
                WorkflowException.dolog(e);
            }
        }
    }

    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        @Qualifier("executor")
        protected ExecutorService executorService;

        @Autowired
        protected CliSubFetcher cliSubFetcher;

        @Bean
        @ConditionalOnMissingBean(value = ClientTokenNotifier.class)
        public ClientTokenNotifier clientTokenNotifier() throws Exception {
            ClientTokenNotifier clientTokenNotifier = new ClientTokenNotifier();
            BeanUtils.copyProperties(this, clientTokenNotifier);
            log.info("ClientTokenNotifier inited");
            return clientTokenNotifier;
        }
    }
}
