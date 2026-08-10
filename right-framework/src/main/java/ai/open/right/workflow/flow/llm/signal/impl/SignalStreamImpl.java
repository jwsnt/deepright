package ai.open.right.workflow.flow.llm.signal.impl;

import ai.open.right.workflow.flow.llm.Message;
import ai.open.right.workflow.flow.llm.signal.*;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.util.Assert;

import java.util.ArrayList;
import java.util.List;

@Slf4j
@Setter
@Getter
public class SignalStreamImpl implements SignalStream, SignalRemoval {

    protected static final String SIGNAL_PATTERN = "\\$\\{.*?\\}";

    protected static final String SIGNAL_B = "${";

    protected static final String SIGNAL_E = "}";

    protected final SignalDistributor signalDistributor;

    protected final SignalConfig signalConfig;

    protected List<String> synthesizer;

    public SignalStreamImpl(SignalConfig signalConfig, SignalDistributor signalDistributor) {
        this.synthesizer = (this.signalConfig = signalConfig).hasSynthesizer() ? new ArrayList<String>() : null;
        this.signalDistributor = signalDistributor;
    }

    public SignalStreamImpl() {
        this.signalDistributor = null;
        this.signalConfig = null;
    }

    @Override
    public String remove(String source) {
        // 本身已经转义，不能二次转义
        return source.replaceAll(SignalStreamImpl.SIGNAL_PATTERN, "");
    }

    @Override
    public void signal(SignalExecutor signalExecutor, Message message) throws Exception {
        int startIndex = 0;
        while ((startIndex = signalExecutor.indexOfContentBuffer(SignalStreamImpl.SIGNAL_B, startIndex)) != -1) {
            int endIndex = signalExecutor.indexOfContentBuffer(SignalStreamImpl.SIGNAL_E, startIndex);
            if (endIndex == -1) {
                break;
            }
            String signalString = signalExecutor.getAndDelContentBuffer(startIndex, endIndex + 1);
            String signalActual = signalString.substring(SignalStreamImpl.SIGNAL_B.length(), signalString.length() - SignalStreamImpl.SIGNAL_E.length());
            signalExecutor.setSignalMetadata(signalActual);
            if (this.synthesizer == null) {
                this.signalDistributor.distribute(this.signalConfig, signalActual, message);
            } else {
                this.synthesizer.add(signalActual);
            }
        }
        signalExecutor.silent(this.signalConfig.getSilent());
        signalExecutor.notify(signalExecutor.indexOfContentBuffer(SignalStreamImpl.SIGNAL_B) == -1);
    }

    public void finish(Message message) throws Exception {
        // Aggregate
        if (this.synthesizer != null) {
            Assert.notEmpty(this.synthesizer, "SignalStreamImp's synthesizer must not be empty");
            this.signalDistributor.distribute(this.signalConfig, this.synthesizer, message);
        }
    }
}