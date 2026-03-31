import * as TabsPrimitive from "@radix-ui/react-tabs";
import { cn } from "../../lib/utils";

function Tabs(props) {
  return <TabsPrimitive.Root className={cn("ui-tabs", props.className)} {...props} />;
}

function TabsList({ className, ...props }) {
  return <TabsPrimitive.List className={cn("ui-tabs__list", className)} {...props} />;
}

function TabsTrigger({ className, ...props }) {
  return <TabsPrimitive.Trigger className={cn("ui-tabs__trigger", className)} {...props} />;
}

function TabsContent({ className, ...props }) {
  return <TabsPrimitive.Content className={cn("ui-tabs__content", className)} {...props} />;
}

export { Tabs, TabsList, TabsTrigger, TabsContent };
