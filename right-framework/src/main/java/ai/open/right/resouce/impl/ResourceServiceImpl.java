package ai.open.right.resouce.impl;

import ai.open.right.WorkflowException;
import ai.open.right.resouce.ResourceService;
import jakarta.annotation.PostConstruct;
import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.BeanUtils;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.autoconfigure.condition.ConditionalOnMissingBean;
import org.springframework.context.ApplicationContext;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.core.io.Resource;
import org.springframework.core.io.support.PathMatchingResourcePatternResolver;
import org.springframework.util.Assert;
import org.springframework.util.ClassUtils;

import java.net.URL;

@Getter
@Setter
@Slf4j
public class ResourceServiceImpl implements ResourceService {

    protected PathMatchingResourcePatternResolver resourceResolver;

    protected ApplicationContext applicationContext;

    protected Class<?> rootClass;

    @PostConstruct
    public void init() throws Exception {
        String[] bean = this.applicationContext.getBeanNamesForAnnotation(SpringBootApplication.class);
        Assert.isTrue(bean.length == 1, "The @SpringBootApplication annotation is present in multiple locations within the project");
        this.resourceResolver = new PathMatchingResourcePatternResolver((this.rootClass = ClassUtils.getUserClass(this.applicationContext.getBean(bean[0]))).getClassLoader());
    }

    @Override
    public URL url(String location) throws Exception {
        Resource[] resources = this.resourceResolver.getResources(location);
        for (Resource res : resources) {
            if (res.isReadable() && res.exists()) {
                return res.getURL();
            }
        }
        throw new WorkflowException(this.buildMessage(location));
    }

    @Override
    public Class<?> root() throws Exception {
        return this.rootClass;
    }

    protected String buildMessage(String location) throws Exception {
        return "The resource can not be find: " + location;
    }
    @Configuration
    @Setter
    @Getter
    public static class InitConfig {

        @Autowired
        protected ApplicationContext applicationContext;

        @Bean
        @ConditionalOnMissingBean(value = ResourceService.class)
        public ResourceService resourceService() throws Exception {
            ResourceServiceImpl resourceServiceImpl = new ResourceServiceImpl();
            BeanUtils.copyProperties(this, resourceServiceImpl);
            log.info("ResourceServiceImpl inited");
            return resourceServiceImpl;
        }
    }
}