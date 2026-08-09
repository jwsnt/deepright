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
import ai.open.right.utils.SplitUtils;
import ai.open.right.workflow.flow.WorkflowTask;
import ai.open.right.workflow.flow.llm.provider.ProviderRequest;
import ai.open.right.workflow.flow.llm.token.TokenData;
import lombok.Builder;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.beans.factory.annotation.Qualifier;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

import java.util.concurrent.ExecutorService;

@Getter
@Setter
@Slf4j
public class ClientTokenNotifier implements TokenNotifier, TokenSource {

    public static final String LANG_KEY_TOKEN_MULTI_OUTPUT = "token.multi.output";

    public static final String LANG_KEY_TOKEN_MULTI_INPUT = "token.multi.input";

    public static final String LANG_KEY_TOKEN_KNOWLEDGE = "token.knowledge";

    public static final String LANG_KEY_TOKEN_PROFILE = "token.profile";

    public static final String LANG_KEY_TOKEN_SKILL = "token.skill";

    public static final String LANG_KEY_TOKEN_CRON = "token.cron";

    public static final String LANG_KEY_TOKEN_TASK = "token.task";

    public static final String LANG_KEY_TOKEN_TEST = "token.test";

    public static final String LANG_KEY_TOKEN_CHAT = "token.chat";

    protected static final String NAME = "token_notifier_client";

    protected static final String TEST = "test";

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
        if (StringUtils.equalsIgnoreCase(SplitUtils.join(workTask), "media@image_gen")) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_MULTI_OUTPUT);
        } else if (StringUtils.equalsIgnoreCase(SplitUtils.join(workTask), "media@ocr_gen")) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_MULTI_INPUT);
        } else if (FeatureFlag.isKnowledgeCommit(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_KNOWLEDGE);
        } else if (FeatureFlag.isProfileCommit(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_PROFILE);
        } else if (FeatureFlag.isSkillExtract(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_SKILL);
        } else if (FeatureFlag.isTask(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_TASK);
        } else if (FeatureFlag.isCron(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_CRON);
        } else if (FeatureFlag.isTest(workTask)) {
            return XmlResourceLang.get(ClientTokenNotifier.LANG_KEY_TOKEN_TEST);
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
                String agentId = !FeatureFlag.isTest(this.providerRequest.getMessage()) ? FeatureUtils.buildAgentId(this.providerRequest.getMessage()) : ClientTokenNotifier.TEST;
                String function = this.tokenSource.source(this.providerRequest.getMessage());
                String app = FeatureUtils.buildApp(this.providerRequest.getMessage());
                Integer thinking = this.tokenData.getThinking();
                String model = this.providerRequest.getModel();
                Integer cache = this.tokenData.getCache();
                Integer input = this.tokenData.getInput();
                Integer total = this.tokenData.getTotal();
                WorkflowException.checkCondition(StringUtils.isEmpty(function), "The function can not be empty");
                WorkflowException.checkCondition(StringUtils.isEmpty(agentId), "The agent id can not be empty");
                WorkflowException.checkCondition(StringUtils.isEmpty(model), "The model can not be empty");
                WorkflowException.checkCondition(StringUtils.isEmpty(app), "The app can not be empty");
                WorkflowException.checkCondition(thinking == null, "The thinking can not be empty");
                WorkflowException.checkCondition(input == null, "The input can not be empty");
                WorkflowException.checkCondition(total == null, "The total can not be empty");
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
                        .echo(false)
                        .build(), buffer.toString(), "");
                WorkflowException.checkCondition(!(pubData.isOk()), pubData.getCmd());
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
