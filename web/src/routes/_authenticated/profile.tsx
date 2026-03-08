import { useState } from "react";
import { createFileRoute } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/lib/auth";
import { useGetProfile, useUpdateProfile, getGetProfileQueryKey } from "@/api/profile/profile";
import { useUpdatePassword } from "@/api/users/users";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";

const ProfilePage = () => {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { data: profile, isLoading } = useGetProfile();
  const queryClient = useQueryClient();
  const updateMutation = useUpdateProfile();
  const passwordMutation = useUpdatePassword();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [profileInitialized, setProfileInitialized] = useState(false);

  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [profileSuccess, setProfileSuccess] = useState(false);
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  // Initialize form fields when profile loads
  if (profile && !profileInitialized) {
    setName(profile.name ?? "");
    setEmail(profile.email ?? "");
    setProfileInitialized(true);
  }

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setProfileSuccess(false);
    updateMutation.mutate(
      { data: { name, email } },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: getGetProfileQueryKey() });
          setProfileSuccess(true);
        },
      },
    );
  };

  const handlePasswordSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setPasswordSuccess(false);
    if (!user?.user_id) return;
    passwordMutation.mutate(
      {
        id: user.user_id,
        data: { current_password: currentPassword, new_password: newPassword },
      },
      {
        onSuccess: () => {
          setCurrentPassword("");
          setNewPassword("");
          setPasswordSuccess(true);
        },
      },
    );
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">{t("profile.title")}</h1>
        <div className="mt-1 flex items-center gap-2">
          <span className="text-muted-foreground">{profile?.email}</span>
          <Badge variant="secondary">{t(`roles.${profile?.role ?? "viewer"}`)}</Badge>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>{t("profile.editProfile")}</CardTitle>
          <CardDescription>{t("common.name")} &amp; {t("common.email")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleProfileSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="profile-name">{t("common.name")}</Label>
              <Input
                id="profile-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="profile-email">{t("common.email")}</Label>
              <Input
                id="profile-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            {updateMutation.isError && (
              <p className="text-sm text-destructive">
                {t("profile.updateError")}
              </p>
            )}
            {profileSuccess && (
              <p className="text-sm text-green-600 dark:text-green-400">
                {t("profile.saveSuccess")}
              </p>
            )}
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? t("common.loading") : t("profile.editProfile")}
            </Button>
          </form>
        </CardContent>
      </Card>

      <Separator />

      <Card>
        <CardHeader>
          <CardTitle>{t("profile.changePassword")}</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="current-password">{t("profile.currentPassword")}</Label>
              <Input
                id="current-password"
                type="password"
                value={currentPassword}
                onChange={(e) => setCurrentPassword(e.target.value)}
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="new-password">{t("profile.newPassword")}</Label>
              <Input
                id="new-password"
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                required
                minLength={8}
              />
            </div>
            {passwordMutation.isError && (
              <p className="text-sm text-destructive">
                {t("profile.passwordError")}
              </p>
            )}
            {passwordSuccess && (
              <p className="text-sm text-green-600 dark:text-green-400">
                {t("profile.saveSuccess")}
              </p>
            )}
            <Button type="submit" disabled={passwordMutation.isPending}>
              {passwordMutation.isPending ? t("common.loading") : t("profile.changePassword")}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
};

export const Route = createFileRoute("/_authenticated/profile")({
  component: ProfilePage,
});
