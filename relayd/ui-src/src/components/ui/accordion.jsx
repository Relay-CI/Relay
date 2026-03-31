import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { cn } from "../../lib/utils";

const Accordion = AccordionPrimitive.Root;

function AccordionItem({ className, ...props }) {
  return <AccordionPrimitive.Item className={cn("ui-accordion__item", className)} {...props} />;
}

function AccordionTrigger({ className, children, ...props }) {
  return (
    <AccordionPrimitive.Header className="ui-accordion__header">
      <AccordionPrimitive.Trigger className={cn("ui-accordion__trigger", className)} {...props}>
        {children}
      </AccordionPrimitive.Trigger>
    </AccordionPrimitive.Header>
  );
}

function AccordionContent({ className, ...props }) {
  return <AccordionPrimitive.Content className={cn("ui-accordion__content", className)} {...props} />;
}

export { Accordion, AccordionItem, AccordionTrigger, AccordionContent };
