package ai.deepright.config.register;

import org.springframework.beans.factory.annotation.AnnotatedBeanDefinition;
import org.springframework.beans.factory.config.BeanDefinition;
import org.springframework.beans.factory.support.BeanDefinitionRegistry;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.EnvironmentAware;
import org.springframework.context.ResourceLoaderAware;
import org.springframework.context.annotation.AnnotationBeanNameGenerator;
import org.springframework.context.annotation.ClassPathScanningCandidateComponentProvider;
import org.springframework.context.annotation.Configuration;
import org.springframework.context.annotation.ImportBeanDefinitionRegistrar;
import org.springframework.core.env.Environment;
import org.springframework.core.io.ResourceLoader;
import org.springframework.core.type.AnnotationMetadata;
import org.springframework.core.type.filter.AnnotationTypeFilter;

public class CustomerConfigurationRegistrar implements ImportBeanDefinitionRegistrar, EnvironmentAware, ResourceLoaderAware {

    protected static final String AUTO_CONFIGURATION_CLASS = CustomerAutoConfiguration.class.getName();

    protected static final String BASE_PACKAGE = "ai.deepright";

    protected final AnnotationBeanNameGenerator beanNameGenerator = new AnnotationBeanNameGenerator();

    protected ResourceLoader resourceLoader;

    protected Environment environment;

    @Override
    public void registerBeanDefinitions(AnnotationMetadata importingClassMetadata, BeanDefinitionRegistry registry) {
        // Register local @Configuration classes early so right-starter fallback beans can defer.
        ClassPathScanningCandidateComponentProvider scanner = new ClassPathScanningCandidateComponentProvider(false, this.environment);
        scanner.setResourceLoader(this.resourceLoader);
        scanner.addIncludeFilter(new AnnotationTypeFilter(Configuration.class));
        scanner.addExcludeFilter(new AnnotationTypeFilter(SpringBootApplication.class));
        for (BeanDefinition candidate : scanner.findCandidateComponents(CustomerConfigurationRegistrar.BASE_PACKAGE)) {
            if (!(candidate instanceof AnnotatedBeanDefinition)) {
                continue;
            }
            String beanClassName = candidate.getBeanClassName();
            if (beanClassName == null || CustomerConfigurationRegistrar.AUTO_CONFIGURATION_CLASS.equals(beanClassName) || isRegistered(registry, beanClassName)) {
                continue;
            }
            String beanName = this.beanNameGenerator.generateBeanName((AnnotatedBeanDefinition) candidate, registry);
            if (registry.containsBeanDefinition(beanName)) {
                continue;
            }
            registry.registerBeanDefinition(beanName, candidate);
        }
    }

    protected Boolean isRegistered(BeanDefinitionRegistry registry, String beanClassName) {
        for (String beanName : registry.getBeanDefinitionNames()) {
            BeanDefinition beanDefinition = registry.getBeanDefinition(beanName);
            if (beanClassName.equals(beanDefinition.getBeanClassName())) {
                return true;
            }
        }
        return false;
    }

    @Override
    public void setResourceLoader(ResourceLoader resourceLoader) {
        this.resourceLoader = resourceLoader;
    }

    @Override
    public void setEnvironment(Environment environment) {
        this.environment = environment;
    }
}
