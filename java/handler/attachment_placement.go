package handler

func javaAttachmentRotation(yaw float32) int {
	return int((yaw+180)*16/360+0.5) & 15
}
